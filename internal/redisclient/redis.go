package redisclient

import (
	"context"
	"time"

	//nolint:depguard
	"github.com/go-redis/redis/v8"
)

type RedisClient interface {
	FlushDB(ctx context.Context) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Incr(ctx context.Context, key string) (int64, error)
	SetTTL(ctx context.Context, key string, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, keys ...string) error
	Decr(ctx context.Context, key string) (int64, error)
	IncrBy(ctx context.Context, key string, value int64) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
	SMembers(ctx context.Context, key string) ([]string, error)
	SAdd(ctx context.Context, key string, members ...interface{}) error
}

type Client struct {
	client *redis.Client
}

func NewRedisClient(addr string) *Client {
	return &Client{
		client: redis.NewClient(&redis.Options{
			Addr: addr,
			DB:   1,
		}),
	}
}

func (r *Client) FlushDB(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

func (r *Client) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *Client) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

func (r *Client) SetTTL(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, key, ttl).Err()
}

func (r *Client) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *Client) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

func (r *Client) Decr(ctx context.Context, key string) (int64, error) {
	return r.client.Decr(ctx, key).Result()
}

func (r *Client) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return r.client.IncrBy(ctx, key, value).Result()
}

func (r *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

func (r *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	return r.client.SMembers(ctx, key).Result()
}

func (r *Client) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return r.client.SAdd(ctx, key, members...).Err()
}
