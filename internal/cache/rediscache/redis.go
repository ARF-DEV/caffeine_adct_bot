package rediscache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ARF-DEV/caffeine_adct_bot/internal/cache"
	"github.com/redis/go-redis/v9"
)

var _ cache.Cache = (*RedisCache)(nil)

type RedisCache struct {
	client *redis.Client
}

func CreateCache(opt *redis.Options) cache.Cache {
	cache := RedisCache{}
	cache.client = redis.NewClient(opt)

	return &cache
}

func (rc *RedisCache) GetAndParse(ctx context.Context, key string, dst interface{}) error {
	res, err := rc.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	if len(res) == 0 {
		return fmt.Errorf("redis key (%s)'s value len is 0", key)
	}
	if err = json.Unmarshal(res, dst); err != nil {
		return err
	}

	return nil
}

func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}) error {

	return rc.SetExp(ctx, key, value, 0)
}

func (rc *RedisCache) SetExp(ctx context.Context, key string, value interface{}, exp time.Duration) error {
	return rc.client.Set(ctx, key, value, exp).Err()
}

func (rc *RedisCache) SetExpFunc(ctx context.Context, key string, value interface{}, expFunc cache.ExpFunc) error {
	return rc.SetExp(ctx, key, value, expFunc(ctx, value))
}
