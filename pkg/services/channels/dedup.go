package channels

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const dedupTTL = 60 * time.Second

// Dedup provides message deduplication using Redis.
type Dedup struct {
	rdb redis.UniversalClient
}

// NewDedup creates a new Dedup instance with the given Redis client.
func NewDedup(rdb redis.UniversalClient) *Dedup {
	return &Dedup{rdb: rdb}
}

// IsDuplicate checks if the message ID has been seen within the TTL window.
// Returns true if this is a duplicate, false if it's a new message.
// Marks the message as seen if it's new.
func (d *Dedup) IsDuplicate(ctx context.Context, ch string, msgID string) (bool, error) {
	if msgID == "" {
		return false, nil
	}

	key := fmt.Sprintf("channel:dedup:%s:%s", ch, msgID)

	// Try to set the key only if it doesn't exist (NX)
	added, err := d.rdb.SetNX(ctx, key, "1", dedupTTL).Result()
	if err != nil {
		return false, fmt.Errorf("dedup check: %w", err)
	}

	if !added {
		slog.Debug("channel: duplicate message skipped",
			"channel", ch, "msg_id", msgID)
	}

	return !added, nil
}

// msgDedup is an in-memory alternative for testing without Redis.
type msgDedup struct {
	mu   sync.Mutex
	seen map[string]time.Time
}

// NewMsgDedup creates an in-memory dedup tracker.
func NewMsgDedup() *msgDedup {
	return &msgDedup{seen: make(map[string]time.Time)}
}

// IsDuplicate checks in-memory deduplication.
func (d *msgDedup) IsDuplicate(msgID string) bool {
	if msgID == "" {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	for k, t := range d.seen {
		if now.Sub(t) > dedupTTL {
			delete(d.seen, k)
		}
	}

	if _, exists := d.seen[msgID]; exists {
		return true
	}
	d.seen[msgID] = now
	return false
}
