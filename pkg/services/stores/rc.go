package stores

import (
	"context"
	"sync"

	redis "github.com/redis/go-redis/v9"

	"github.com/liut/morign/pkg/settings"
)

// RedisClient is the Redis client type
type RedisClient = redis.UniversalClient

var (
	rcOnce sync.Once
	rcu    RedisClient
)

// SgtRC start return a singleton instance of redis client
func SgtRC() RedisClient {
	rcOnce.Do(func() {
		redisURI := settings.Current.RedisURI
		opt, err := redis.ParseURL(redisURI)
		if err != nil {
			logger().Panicw("prase redisURI fail", "uri", redisURI, "err", err)
		}
		rcu = redis.NewClient(opt)
		pingStatus := rcu.Ping(context.Background())
		if err = pingStatus.Err(); err != nil {
			logger().Panicw("ping redis fail", "err", err)
		}
	})

	return rcu
}
