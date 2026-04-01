package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/liut/morign/pkg/models/aigc"
	"github.com/liut/morign/pkg/models/channel"
	"github.com/liut/morign/pkg/services/channels"
	"github.com/liut/morign/pkg/services/llm"
	"github.com/liut/morign/pkg/services/stores"
	"github.com/liut/morign/pkg/services/tools"
)

// channelHandler holds dependencies for handling channel messages.
type channelHandler struct {
	sto      stores.Storage
	llm      llm.Client
	toolreg  *tools.Registry
	toolExec *ToolExecutor
}

// InitChannels initializes channel adapters from preset configuration.
func InitChannels(r chi.Router, preset *aigc.Preset, sto stores.Storage, llmClient llm.Client, toolreg *tools.Registry) error {
	chandler := &channelHandler{
		sto:      sto,
		llm:      llmClient,
		toolreg:  toolreg,
		toolExec: NewToolExecutor(toolreg),
	}

	if preset == nil || len(preset.Channels) == 0 {
		slog.Info("channel: no platforms configured")
		return nil
	}

	for name, cfg := range preset.Channels {
		if !cfg.Enable {
			slog.Debug("channel: skipping disabled channel", "name", name)
			continue
		}

		// Inject mode into config for channel factory
		channelConfig := cfg.Config
		if channelConfig == nil {
			channelConfig = make(map[string]any)
		}
		channelConfig["mode"] = cfg.Mode

		p, err := channels.NewChannel(name, channelConfig)
		if err != nil {
			slog.Warn("channel: create failed", "name", name, "error", err)
			continue
		}

		if err := p.Start(chandler.MessageHandler); err != nil {
			slog.Warn("channel: start failed", "name", name, "error", err)
			continue
		}

		// Register HTTP routes if channel supports webhook callback
		if httpRouter, ok := p.(channels.HTTPRouter); ok {
			callbackPath, _ := channelConfig["callback_path"].(string)
			if callbackPath == "" {
				callbackPath = "/" + name + "/callback"
			}
			httpRouter.RegisterHTTPRoutes(r, callbackPath, chandler.MessageHandler)
			slog.Info("channel: HTTP routes registered", "name", name, "path", callbackPath)
		}

		// Use name + mode as unique key to support multiple instances of same channel type
		key := name
		if cfg.Mode != "" {
			key = name + "-" + cfg.Mode
		}
		channels.TrackChannel(key, p)
		slog.Info("channel: started", "name", name, "mode", cfg.Mode, "key", key)
	}

	slog.Info("channel: manager initialized")
	return nil
}

// StopChannels stops all channel adapters.
func StopChannels() {
	channels.StopAll()
}

// MessageHandler processes incoming messages from channel adapters.
func (chh *channelHandler) MessageHandler(p channel.Channel, msg *channel.Message) {
	if chh == nil {
		slog.Error("channel: handler not initialized")
		return
	}

	ctx := context.Background()
	user, err := chh.sto.Convo().GetUserWith(ctx, msg.UserID)
	if err == nil {
		logger().Debugw("found user", "id", user.ID, "userID", msg.UserID)
		ctx = ContextWithUser(ctx, user)
		if token, err := stores.LoadTokenWithUser(ctx, user.StringID()); err == nil {
			ctx = stores.OAuthContextWithToken(ctx, token)
		}
	} else {
		logger().Infow("not found user", "userID", msg.UserID, "err", err)
	}

	// Build the chat request
	cs := stores.GetOrCreateConversationBySessionKey(ctx, msg.SessionKey)

	slog.Info("channel: message received",
		"channel", p.Name(),
		"session", msg.SessionKey,
		"conversation", cs.GetID(),
		"user", msg.UserID,
		"content_len", len(msg.Content),
	)

	// Prepare system message and tools
	sysMsg, tools := prepareSystemMessage(ctx, chh.sto, chh.toolreg, msg.Content, cs)

	// Build user message with any attachments
	content := msg.Content
	if len(msg.Images) > 0 {
		content += "\n[User sent an image]"
	}
	if msg.Audio != nil {
		content += "\n[User sent a voice message]"
	}

	// Load conversation history
	messages := []llm.Message{sysMsg}
	history, _ := cs.ListHistory(ctx)
	for _, hi := range history {
		if hi.ChatItem != nil {
			if hi.ChatItem.User != "" {
				messages = append(messages, llm.Message{Role: llm.RoleUser, Content: hi.ChatItem.User})
			}
			if hi.ChatItem.Assistant != "" {
				messages = append(messages, llm.Message{Role: llm.RoleAssistant, Content: hi.ChatItem.Assistant})
			}
		}
	}
	messages = append(messages, llm.Message{
		Role:    llm.RoleUser,
		Content: content,
	})

	// Execute the chat with tool call loop
	exec := func(ctx context.Context, messages []llm.Message, tools []llm.ToolDefinition) (string, []llm.ToolCall, *llm.Usage, error) {
		result, err := chh.llm.Chat(ctx, messages, tools)
		if err != nil {
			return "", nil, nil, err
		}
		return result.Content, result.ToolCalls, result.Usage, nil
	}
	answer, _, _, err := chh.executeToolCallLoop(ctx, messages, tools, exec)
	if err != nil {
		slog.Error("channel: chat execution failed",
			"channel", p.Name(), "error", err)
		channelReplyError(p, msg, "AI processing failed")
		return
	}

	// Save to history (only final answer, not tool call content)
	if len(answer) > 0 {
		hi := &aigc.HistoryItem{
			Time: time.Now().Unix(),
			UID:  msg.UserID,
			ChatItem: &aigc.HistoryChatItem{
				User:      msg.Content,
				Assistant: answer,
			},
		}
		if err := cs.AddHistory(ctx, hi); err == nil {
			if err := cs.Save(ctx); err != nil {
				slog.Warn("channel: save history failed", "err", err)
			}
		}
	}

	// Send reply to channel
	if err := p.Reply(ctx, msg.ReplyCtx, answer); err != nil {
		slog.Error("channel: reply failed",
			"channel", p.Name(), "error", err)
	}
}

// executeToolCallLoop executes tool calls in a loop until no more tool calls
func (chh *channelHandler) executeToolCallLoop(ctx context.Context, messages []llm.Message, tools []llm.ToolDefinition, exec chatExecutor) (string, []llm.ToolCall, *llm.Usage, error) {
	return chh.toolExec.ExecuteToolCallLoop(ctx, messages, tools, exec)
}

// channelReplyError sends an error message back to the channel.
func channelReplyError(p channel.Channel, msg *channel.Message, errorText string) {
	ctx := context.Background()
	if err := p.Reply(ctx, msg.ReplyCtx, errorText); err != nil {
		slog.Error("channel: send error reply failed",
			"channel", p.Name(), "error", err)
	}
}
