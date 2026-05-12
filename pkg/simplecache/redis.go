package simplecache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	client *redis.Client
	prefix string
}

// NewRedis creates a simplecache.Cache backed by Redis.
// The prefix namespaces keys so multiple caches can share one Redis instance.
func NewRedis(client *redis.Client, prefix string) Cache {
	return &redisCache{
		client: client,
		prefix: prefix,
	}
}

func (c *redisCache) key(k interface{}) string {
	b, _ := json.Marshal(k)
	return c.prefix + ":" + string(b)
}

func (c *redisCache) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Second)
}

func (c *redisCache) Add(k, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	ctx, cancel := c.ctx()
	defer cancel()
	c.client.Set(ctx, c.key(k), b, 0)
}

func (c *redisCache) Get(k interface{}) (interface{}, bool) {
	ctx, cancel := c.ctx()
	defer cancel()
	val, err := c.client.Get(ctx, c.key(k)).Bytes()
	if err != nil {
		return nil, false
	}
	var result interface{}
	if err := json.Unmarshal(val, &result); err != nil {
		return nil, false
	}
	return result, true
}

func (c *redisCache) Remove(k interface{}) (interface{}, bool) {
	val, ok := c.Get(k)
	if !ok {
		return nil, false
	}
	ctx, cancel := c.ctx()
	defer cancel()
	c.client.Del(ctx, c.key(k))
	return val, true
}

func (c *redisCache) Size() int {
	ctx, cancel := c.ctx()
	defer cancel()
	var count int
	iter := c.client.Scan(ctx, 0, c.prefix+":*", 0).Iterator()
	for iter.Next(ctx) {
		count++
	}
	return count
}

func (c *redisCache) Keys() []interface{} {
	ctx, cancel := c.ctx()
	defer cancel()
	var keys []interface{}
	prefixLen := len(c.prefix) + 1
	iter := c.client.Scan(ctx, 0, c.prefix+":*", 0).Iterator()
	for iter.Next(ctx) {
		k := iter.Val()
		if len(k) > prefixLen {
			var parsed interface{}
			if err := json.Unmarshal([]byte(k[prefixLen:]), &parsed); err == nil {
				keys = append(keys, parsed)
			}
		}
	}
	return keys
}
