package grpc_test

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	//nolint:depguard
	"github.com/stretchr/testify/assert"

	//nolint:depguard
	pb "github.com/Dmitry-Tsarkov/brute-force-service-grw/api"
	//nolint:depguard
	"github.com/Dmitry-Tsarkov/brute-force-service-grw/internal/grpc"
	//nolint:depguard
	"github.com/Dmitry-Tsarkov/brute-force-service-grw/internal/redisclient"
)

const loginLimit = 10

func createTestRedisClient() *redisclient.Client {
	return redisclient.NewRedisClient("localhost:6379")
}

func TestCheckAuth_WithinLimit(t *testing.T) {
	redisClient := createTestRedisClient()

	err := redisClient.FlushDB(context.Background())
	if err != nil {
		log.Fatalf("Failed to flush DB: %v", err)
	}

	defer func() {
		if err := redisClient.FlushDB(context.Background()); err != nil {
			log.Printf("Error flushing DB: %v", err)
		}
	}()

	authServer := &grpc.AuthServiceServer{RedisClient: redisClient}

	req := &pb.AuthRequest{
		Login:    "testuser",
		Password: "password123",
		Ip:       "127.0.0.1",
	}

	err = redisClient.Set(context.Background(), "whitelist:127.0.0.1", "true", time.Hour)
	assert.NoError(t, err, "Ошибка при добавлении IP в белый список")

	for i := 0; i < 5; i++ {
		res, err := authServer.CheckAuth(context.Background(), req)
		assert.NoError(t, err)
		assert.True(t, res.Ok, "Авторизация должна быть разрешена")
	}
}

func TestCheckAuth_ExceedsLimit(t *testing.T) {
	redisClient := createTestRedisClient()
	err := redisClient.FlushDB(context.Background())
	if err != nil {
		log.Fatalf("Failed to flush DB: %v", err)
	}
	defer func() {
		if err := redisClient.FlushDB(context.Background()); err != nil {
			log.Printf("Error flushing DB: %v", err)
		}
	}()
	authServer := &grpc.AuthServiceServer{RedisClient: redisClient}
	req := &pb.AuthRequest{
		Login:    "testuser",
		Password: "password123",
		Ip:       "127.0.0.1",
	}
	for i := 0; i < loginLimit+1; i++ {
		res, err := authServer.CheckAuth(context.Background(), req)
		if i < loginLimit {
			assert.NoError(t, err)
			assert.True(t, res.Ok, "Авторизация должна быть разрешена до достижения лимита")
		} else {
			assert.NoError(t, err)
			assert.False(t, res.Ok, "Авторизация должна быть отклонена после достижения лимита")
		}
	}
}

func TestCheckAuth_Whitelist(t *testing.T) {
	redisClient := createTestRedisClient()
	defer func() {
		if err := redisClient.FlushDB(context.Background()); err != nil {
			log.Printf("Error flushing DB: %v", err)
		}
	}()

	authServer := &grpc.AuthServiceServer{RedisClient: redisClient}
	err := redisClient.Set(context.Background(), "whitelist:127.0.0.1", "true", time.Hour)
	assert.NoError(t, err)

	req := &pb.AuthRequest{
		Login:    "testuser",
		Password: "password123",
		Ip:       "127.0.0.1",
	}
	res, err := authServer.CheckAuth(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, res.Ok, "Авторизация должна быть разрешена для IP в белом списке")
}

func TestCheckAuth_Blacklist(t *testing.T) {
	redisClient := createTestRedisClient()

	defer func() {
		if err := redisClient.FlushDB(context.Background()); err != nil {
			log.Printf("Error flushing DB: %v", err)
		}
	}()

	authServer := &grpc.AuthServiceServer{RedisClient: redisClient}

	err := redisClient.Set(context.Background(), "blacklist:127.0.0.1", "true", time.Hour)
	assert.NoError(t, err)

	req := &pb.AuthRequest{
		Login:    "testuser",
		Password: "password123",
		Ip:       "127.0.0.1",
	}
	res, err := authServer.CheckAuth(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, res.Ok, "Авторизация должна быть отклонена для IP в черном списке")
}

func TestResetBucket(t *testing.T) {
	redisClient := createTestRedisClient()
	defer func() {
		if err := redisClient.FlushDB(context.Background()); err != nil {
			log.Printf("Error flushing DB: %v", err)
		}
	}()

	authServer := &grpc.AuthServiceServer{RedisClient: redisClient}
	err := redisClient.Set(context.Background(), "whitelist:127.0.0.1", "true", time.Hour)
	assert.NoError(t, err)
	for i := 0; i < 3; i++ {
		_, _ = authServer.CheckAuth(context.Background(), &pb.AuthRequest{
			Login:    "testuser",
			Password: "password123",
			Ip:       "127.0.0.1",
		})
	}
	_, err = authServer.ResetBucket(context.Background(), &pb.ResetRequest{
		Login: "testuser",
		Ip:    "127.0.0.1",
	})
	assert.NoError(t, err)
	resp, err := authServer.CheckAuth(context.Background(), &pb.AuthRequest{
		Login:    "testuser",
		Password: "password123",
		Ip:       "127.0.0.1",
	})
	assert.NoError(t, err)
	assert.True(t, resp.Ok, "Авторизация должна быть разрешена после сброса")
}

func TestCheckAuth_RedisUnavailable(t *testing.T) {
	s := &grpc.AuthServiceServer{
		RedisClient: redisclient.NewRedisClient("localhost:9999"),
	}
	resp, err := s.CheckAuth(context.Background(), &pb.AuthRequest{
		Login:    "testuser",
		Password: "password123",
		Ip:       "127.0.0.1",
	})
	assert.Error(t, err, "Должна возникнуть ошибка при недоступности Redis")
	assert.False(t, resp.Ok, "Авторизация должна быть запрещена при недоступности Redis")
}

func TestCheckAuth_AtLimit(t *testing.T) {
	redisClient := createTestRedisClient()
	err := redisClient.FlushDB(context.Background())
	if err != nil {
		log.Fatalf("Failed to flush DB: %v", err)
	}
	defer func() {
		if err := redisClient.FlushDB(context.Background()); err != nil {
			log.Printf("Error flushing DB: %v", err)
		}
	}()

	authServer := &grpc.AuthServiceServer{RedisClient: redisClient}
	req := &pb.AuthRequest{
		Login:    "testuser",
		Password: "password123",
		Ip:       "127.0.0.1",
	}

	for i := 0; i < loginLimit; i++ {
		res, err := authServer.CheckAuth(context.Background(), req)
		assert.NoError(t, err)
		assert.True(t, res.Ok, "Авторизация должна быть разрешена до достижения лимита")
	}

	res, err := authServer.CheckAuth(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, res.Ok, "Авторизация должна быть отклонена после достижения лимита")
}

func TestCheckAuth_Parallel(t *testing.T) {
	redisClient := createTestRedisClient()
	err := redisClient.FlushDB(context.Background())
	if err != nil {
		log.Fatalf("Failed to flush DB: %v", err)
	}
	defer func() {
		if err := redisClient.FlushDB(context.Background()); err != nil {
			log.Printf("Error flushing DB: %v", err)
		}
	}()

	authServer := &grpc.AuthServiceServer{RedisClient: redisClient}
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			req := &pb.AuthRequest{
				Login:    fmt.Sprintf("testuser%d", i),
				Password: "password123",
				Ip:       fmt.Sprintf("127.0.0.%d", i),
			}
			log.Printf("Проверка белого списка для IP: %s", req.Ip)

			res, err := authServer.CheckAuth(context.Background(), req)
			if err != nil {
				t.Errorf("Ошибка при авторизации: %v", err)
				return
			}
			if res.Ok {
				assert.True(t, res.Ok, "Авторизация должна быть разрешена в параллельном режиме")
			} else {
				assert.False(t, res.Ok, "Авторизация должна быть отклонена после достижения лимита в параллельном режиме")
			}
		}(i)
	}

	wg.Wait()
}

func TestCheckAuth_InvalidData(t *testing.T) {
	redisClient := createTestRedisClient()
	err := redisClient.FlushDB(context.Background())
	if err != nil {
		log.Fatalf("Failed to flush DB: %v", err)
	}
	defer func() {
		if err := redisClient.FlushDB(context.Background()); err != nil {
			log.Printf("Error flushing DB: %v", err)
		}
	}()

	authServer := &grpc.AuthServiceServer{RedisClient: redisClient}
	req := &pb.AuthRequest{}

	res, err := authServer.CheckAuth(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, res.Ok, "Авторизация должна быть отклонена для пустого запроса")
	req = &pb.AuthRequest{
		Login:    "",
		Password: "",
		Ip:       "",
	}

	res, err = authServer.CheckAuth(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, res.Ok, "Авторизация должна быть отклонена для неверных данных")
}
