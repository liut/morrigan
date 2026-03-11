package stores

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/liut/morign/pkg/models/aigc"
	"github.com/liut/morign/pkg/models/convo"
	"github.com/redis/go-redis/v9"
)

// newTestConversation 创建用于测试的 Conversation（不依赖数据库）
func newTestConversation(t *testing.T) (*miniredis.Miniredis, Conversation) {
	t.Helper()

	// 启动 miniredis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	// 创建连接到 miniredis 的 Redis 客户端
	rc := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 直接创建 conversation 结构，跳过数据库
	sess := convo.NewSessionWithID("test-" + t.Name())
	conv := &conversation{
		id:   sess.ID,
		rc:   rc,
		sess: sess,
		sto:  nil, // 测试不依赖数据库
	}

	return mr, conv
}

func TestAddHistory(t *testing.T) {
	mr, conv := newTestConversation(t)
	defer mr.Close()

	ctx := context.Background()

	// 测试添加单条消息
	item := &aigc.HistoryItem{
		Time: 1234567890,
		ChatItem: &aigc.HistoryChatItem{
			User: "Hello",
		},
		UID: "user1",
	}

	err := conv.AddHistory(ctx, item)
	if err != nil {
		t.Fatalf("AddHistory failed: %v", err)
	}

	// 验证添加成功
	history, err := conv.ListHistory(ctx)
	if err != nil {
		t.Fatalf("ListHistory failed: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("expected 1 item, got %d", len(history))
	}
	if history[0].ChatItem == nil || history[0].ChatItem.User != "Hello" {
		t.Errorf("expected user 'Hello', got %v", history[0].ChatItem)
	}
}

func TestAddHistory_Multiple(t *testing.T) {
	mr, conv := newTestConversation(t)
	defer mr.Close()

	ctx := context.Background()

	// 添加多条消息
	items := []*aigc.HistoryItem{
		{Time: 1, ChatItem: &aigc.HistoryChatItem{User: "First"}},
		{Time: 2, ChatItem: &aigc.HistoryChatItem{User: "Second"}},
		{Time: 3, ChatItem: &aigc.HistoryChatItem{User: "Third"}},
	}

	for _, item := range items {
		if err := conv.AddHistory(ctx, item); err != nil {
			t.Fatalf("AddHistory failed: %v", err)
		}
	}

	history, err := conv.ListHistory(ctx)
	if err != nil {
		t.Fatalf("ListHistory failed: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("expected 3 items, got %d", len(history))
	}
}

func TestAddHistory_DuplicateWithChatItem(t *testing.T) {
	mr, conv := newTestConversation(t)
	defer mr.Close()

	ctx := context.Background()

	// 添加第一条消息（使用 ChatItem）
	item1 := &aigc.HistoryItem{
		Time: 1234567890,
		ChatItem: &aigc.HistoryChatItem{
			User: "What's the weather?",
		},
	}
	if err := conv.AddHistory(ctx, item1); err != nil {
		t.Fatalf("AddHistory failed: %v", err)
	}

	// 添加重复消息（相同 ChatItem.User）
	item2 := &aigc.HistoryItem{
		Time: 1234567891,
		ChatItem: &aigc.HistoryChatItem{
			User: "What's the weather?",
		},
	}
	if err := conv.AddHistory(ctx, item2); err != nil {
		t.Fatalf("AddHistory failed: %v", err)
	}

	// 验证只有一条消息（去重生效）
	history, err := conv.ListHistory(ctx)
	if err != nil {
		t.Fatalf("ListHistory failed: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("expected 1 item (duplicate should be skipped), got %d", len(history))
	}
}

func TestAddHistory_DifferentContent(t *testing.T) {
	mr, conv := newTestConversation(t)
	defer mr.Close()

	ctx := context.Background()

	// 添加第一条消息
	item1 := &aigc.HistoryItem{
		Time: 1234567890,
		ChatItem: &aigc.HistoryChatItem{
			User: "Hello",
		},
	}
	if err := conv.AddHistory(ctx, item1); err != nil {
		t.Fatalf("AddHistory failed: %v", err)
	}

	// 添加不同消息（应该添加，不去重）
	item2 := &aigc.HistoryItem{
		Time: 1234567891,
		ChatItem: &aigc.HistoryChatItem{
			User: "World",
		},
	}
	if err := conv.AddHistory(ctx, item2); err != nil {
		t.Fatalf("AddHistory failed: %v", err)
	}

	// 验证有两条消息
	history, err := conv.ListHistory(ctx)
	if err != nil {
		t.Fatalf("ListHistory failed: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("expected 2 items, got %d", len(history))
	}
}

func TestListHistory(t *testing.T) {
	mr, conv := newTestConversation(t)
	defer mr.Close()

	ctx := context.Background()

	// 空列表
	history, err := conv.ListHistory(ctx)
	if err != nil {
		t.Fatalf("ListHistory failed: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("expected empty list, got %d items", len(history))
	}

	// 添加消息后再查询
	for i := 1; i <= 5; i++ {
		item := &aigc.HistoryItem{
			Time: int64(i * 1000),
			ChatItem: &aigc.HistoryChatItem{
				User: "Message " + string(rune('0'+i)),
			},
		}
		if err := conv.AddHistory(ctx, item); err != nil {
			t.Fatalf("AddHistory failed: %v", err)
		}
	}

	history, err = conv.ListHistory(ctx)
	if err != nil {
		t.Fatalf("ListHistory failed: %v", err)
	}
	if len(history) != 5 {
		t.Errorf("expected 5 items, got %d", len(history))
	}
}

func TestClearHistory(t *testing.T) {
	mr, conv := newTestConversation(t)
	defer mr.Close()

	ctx := context.Background()

	// 添加一些消息
	for i := 1; i <= 3; i++ {
		item := &aigc.HistoryItem{
			Time: int64(i * 1000),
			ChatItem: &aigc.HistoryChatItem{
				User: "Message " + string(rune('0'+i)),
			},
		}
		if err := conv.AddHistory(ctx, item); err != nil {
			t.Fatalf("AddHistory failed: %v", err)
		}
	}

	// 验证有消息
	history, err := conv.ListHistory(ctx)
	if err != nil {
		t.Fatalf("ListHistory failed: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("expected 3 items, got %d", len(history))
	}

	// 清除历史
	err = conv.ClearHistory(ctx)
	if err != nil {
		t.Fatalf("ClearHistory failed: %v", err)
	}

	// 验证已清除
	history, err = conv.ListHistory(ctx)
	if err != nil {
		t.Fatalf("ListHistory failed: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("expected 0 items after clear, got %d", len(history))
	}
}

func TestHistoryMaxLength(t *testing.T) {
	mr, conv := newTestConversation(t)
	defer mr.Close()

	ctx := context.Background()

	// 添加超过 historyMaxLength (25) 条消息
	for i := 1; i <= 30; i++ {
		item := &aigc.HistoryItem{
			Time: int64(i * 1000),
			ChatItem: &aigc.HistoryChatItem{
				User: "Message " + string(rune('0'+i%10)),
			},
		}
		if err := conv.AddHistory(ctx, item); err != nil {
			t.Fatalf("AddHistory failed: %v", err)
		}
	}

	// 验证历史被限制在 max length
	history, err := conv.ListHistory(ctx)
	if err != nil {
		t.Fatalf("ListHistory failed: %v", err)
	}
	if len(history) != historyMaxLength {
		t.Errorf("expected %d items, got %d", historyMaxLength, len(history))
	}
}

func TestCountHistory(t *testing.T) {
	mr, conv := newTestConversation(t)
	defer mr.Close()

	ctx := context.Background()

	// 空列表计数
	count := conv.CountHistory(ctx)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	// 添加消息后计数
	items := []*aigc.HistoryItem{
		{Time: 1, ChatItem: &aigc.HistoryChatItem{User: "First"}},
		{Time: 2, ChatItem: &aigc.HistoryChatItem{User: "Second"}},
		{Time: 3, ChatItem: &aigc.HistoryChatItem{User: "Third"}},
	}
	for _, item := range items {
		if err := conv.AddHistory(ctx, item); err != nil {
			t.Fatalf("AddHistory failed: %v", err)
		}
	}

	count = conv.CountHistory(ctx)
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}

	// 清除后计数
	err := conv.ClearHistory(ctx)
	if err != nil {
		t.Fatalf("ClearHistory failed: %v", err)
	}
	count = conv.CountHistory(ctx)
	if count != 0 {
		t.Errorf("expected 0 after clear, got %d", count)
	}
}
