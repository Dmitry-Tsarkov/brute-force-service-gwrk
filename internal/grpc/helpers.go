package grpc

import (
	"context"
	"errors"
	"log"
	"time"

	//nolint:depguard
	"github.com/go-redis/redis/v8"
)

var redisClient *redis.Client

func InitRedisClient() {
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	_, err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
}

func GetBucketState(login string, ip string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	val, err := redisClient.Get(ctx, login+"_"+ip).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	log.Printf("Bucket state: %s", val)
	return true, nil
}
