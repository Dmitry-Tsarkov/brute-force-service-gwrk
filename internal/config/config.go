package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	RedisHost     string
	RedisPort     string
	GRPCPort      string
	LoginLimit    int
	IPLimit       int
	PasswordLimit int
	BucketTTL     time.Duration
}

func LoadConfig() *Config {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}

	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	loginLimit := 10
	ipLimit := 100
	passwordLimit := 5
	bucketTTL := time.Minute

	if limit := os.Getenv("LOGIN_LIMIT"); limit != "" {
		var err error
		loginLimit, err = strconv.Atoi(limit)
		if err != nil {
			log.Printf("Ошибка чтения LOGIN_LIMIT, используем значение по умолчанию %d", loginLimit)
		}
	}

	if limit := os.Getenv("IP_LIMIT"); limit != "" {
		var err error
		ipLimit, err = strconv.Atoi(limit)
		if err != nil {
			log.Printf("Ошибка чтения IP_LIMIT, используем значение по умолчанию %d", ipLimit)
		}
	}

	if limit := os.Getenv("PASSWORD_LIMIT"); limit != "" {
		var err error
		passwordLimit, err = strconv.Atoi(limit)
		if err != nil {
			log.Printf("Ошибка чтения PASSWORD_LIMIT, используем значение по умолчанию %d", passwordLimit)
		}
	}

	return &Config{
		RedisHost:     redisHost,
		RedisPort:     redisPort,
		GRPCPort:      grpcPort,
		LoginLimit:    loginLimit,
		IPLimit:       ipLimit,
		PasswordLimit: passwordLimit,
		BucketTTL:     bucketTTL,
	}
}
