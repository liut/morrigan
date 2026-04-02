package api

import (
	"context"
	"strings"

	"github.com/liut/morign/pkg/models/channel"
	"github.com/liut/morign/pkg/services/stores"
)

type Command struct {
	Name    string
	Aliases []string
	Desc    string
	Action  func(ctx context.Context, msg *channel.Message) (bool, error)
}

var commandRegistry = []Command{
	{
		Name:    "reset",
		Aliases: []string{"/reset", "/new", "/clear"},
		Desc:    "重置会话，创建新的 csid",
		Action:  handleResetCommand,
	},
}

func DetectCommand(content string) Command {
	trimmed := strings.TrimSpace(content)
	for _, cmd := range commandRegistry {
		for _, alias := range cmd.Aliases {
			if strings.HasPrefix(trimmed, alias) {
				return cmd
			}
		}
	}
	return Command{}
}

func handleResetCommand(ctx context.Context, msg *channel.Message) (bool, error) {
	if err := stores.ResetSessionBySessionKey(ctx, msg.SessionKey); err != nil {
		return false, err
	}
	logger().Infow("command: session reset", "sessionKey", msg.SessionKey)
	return true, nil
}
