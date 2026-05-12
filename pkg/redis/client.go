package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	pkgSync "github.com/stackrox/rox/pkg/sync"
)

var (
	log      = logging.LoggerForModule()
	once     pkgSync.Once
	instance *redis.Client
)

// ClientSingleton returns a shared Redis client instance.
func ClientSingleton() *redis.Client {
	once.Do(func() {
		endpoint := env.RedisEndpoint.Setting()
		instance = redis.NewClient(&redis.Options{
			Addr: endpoint,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := instance.Ping(ctx).Err(); err != nil {
			log.Warnf("Redis ping failed (endpoint: %s): %v", endpoint, err)
		} else {
			log.Infof("Connected to Redis at %s", endpoint)
		}
	})
	return instance
}
