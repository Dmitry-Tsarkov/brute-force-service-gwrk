package redisclient

import (
	"context"
	"time"

	//nolint:depguard
	"github.com/go-redis/redis/v8"
)

type RedisAdapter struct {
	client *redis.Client
}

func NewRedisAdapter(client *redis.Client) *RedisAdapter {
	return &RedisAdapter{client: client}
}

func (r *RedisAdapter) Decr(ctx context.Context, key string) (int64, error) {
	result := r.client.Decr(ctx, key)
	return result.Result()
}

func (r *RedisAdapter) Get(ctx context.Context, key string) (string, error) {
	result := r.client.Get(ctx, key)
	return result.Result()
}

func (r *RedisAdapter) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisAdapter) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisAdapter) Incr(ctx context.Context, key string) (int64, error) {
	result := r.client.Incr(ctx, key)
	return result.Result()
}

func (r *RedisAdapter) SetTTL(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}
