// file: service/cache.go

package service

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9" // CORRECTED IMPORT PATH
)

// ICacheClient defines the contract for a cache client.
// This abstraction allows us to decouple the AccountService from a concrete
// Redis implementation, enabling easier testing and future flexibility.
type ICacheClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}
