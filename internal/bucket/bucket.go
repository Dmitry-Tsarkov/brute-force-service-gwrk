package bucket

import (
	"context"
	"errors"
	"strconv"
	"time"

	//nolint:depguard
	"github.com/go-redis/redis/v8"
	//nolint:depguard
	"github.com/Dmitry-Tsarkov/brute-force-service-grw/internal/redisclient"
)

type Bucket struct {
	client     redisclient.RedisClient
	key        string
	maxTokens  int
	refillRate int
}

func NewBucket(client redisclient.RedisClient, key string, maxTokens int, refillRate int) *Bucket {
	return &Bucket{
		client:     client,
		key:        key,
		maxTokens:  maxTokens,
		refillRate: refillRate,
	}
}

func (b *Bucket) Allow(ctx context.Context) (bool, error) {
	tokens, err := b.client.Get(ctx, b.key)
	if errors.Is(err, redis.Nil) {
		if err := b.client.Set(ctx, b.key, strconv.Itoa(b.maxTokens), time.Minute*1); err != nil {
			return false, err
		}
		tokens = strconv.Itoa(b.maxTokens)
	} else if err != nil {
		return false, err
	}

	tokenCount, err := strconv.Atoi(tokens)
	if err != nil {
		return false, err
	}

	if tokenCount > 0 {
		_, err := b.client.Decr(ctx, b.key)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func (b *Bucket) Refill(ctx context.Context) error {
	_, err := b.client.IncrBy(ctx, b.key, int64(b.refillRate))
	return err
}
