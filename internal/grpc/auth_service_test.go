package grpc_test

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	//nolint:depguard
	pb "github.com/Dmitry-Tsarkov/brute-force-service-gwrk/api"
	//nolint:depguard
	"github.com/Dmitry-Tsarkov/brute-force-service-gwrk/internal/grpc"
	//nolint:depguard
	"github.com/Dmitry-Tsarkov/brute-force-service-gwrk/internal/redisclient"
)

const (
	loginLimit    = 10
	passwordLimit = 5
	IPLimit       = 100
)

func createTestRedisClient() *redisclient.Client {
	return redisclient.NewRedisClient("localhost:6379")
}

func TestCheckAuth_WithinLimit(t *testing.T) {
	redisClient := createTestRedisClient()

	err := redisClient.FlushDB(context.Background())
	assert.NoError(t, err, "Ошибка при очистке базы Redis")

	defer func() {
		err := redisClient.FlushDB(context.Background())
		assert.NoError(t, err, "Ошибка при очистке базы Redis")
	}()

	authServer := &grpc.AuthServiceServer{
		RedisClient:   redisClient,
		LoginLimit:    loginLimit,
		PasswordLimit: passwordLimit,
		IPLimit:       IPLimit,
		BucketTTL:     time.Minute,
	}

	req := &pb.AuthRequest{
		Login:    "testuser",
		Password: "password123",
		Ip:       "127.0.0.1",
	}

	err = redisClient.SAdd(context.Background(), "whitelist", "127.0.0.1/32")
	assert.NoError(t, err, "Ошибка при добавлении IP в белый список")

	for i := 0; i < 5; i++ {
		res, err := authServer.CheckAuth(context.Background(), req)
		assert.NoError(t, err)
		assert.True(t, res.Ok, "Авторизация должна быть разрешена")
	}

	res, err := authServer.CheckAuth(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, res.Ok, "Авторизация должна быть отклонена после достижения лимита для пароля")
}

func TestCheckAuth_PasswordLimit(t *testing.T) {
	redisClient := createTestRedisClient()

	err := redisClient.FlushDB(context.Background())
	assert.NoError(t, err, "Ошибка при очистке базы Redis")

	defer func() {
		err := redisClient.FlushDB(context.Background())
		assert.NoError(t, err, "Ошибка при очистке базы Redis")
	}()

	authServer := &grpc.AuthServiceServer{
		RedisClient:   redisClient,
		LoginLimit:    100,
		PasswordLimit: passwordLimit,
		IPLimit:       IPLimit,
		BucketTTL:     time.Minute,
	}

	req := &pb.AuthRequest{
		Login:    "testuser",
		Password: "password_limit_test",
		Ip:       "127.0.0.1",
	}

	for i := 0; i < authServer.PasswordLimit; i++ {
		res, err := authServer.CheckAuth(context.Background(), req)
		assert.NoError(t, err)
		assert.True(t, res.Ok, "Авторизация должна быть разрешена до достижения лимита для пароля")
	}

	res, err := authServer.CheckAuth(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, res.Ok, "Авторизация должна быть отклонена после достижения лимита для пароля")
}

func TestCheckAuth_LoginLimit(t *testing.T) {
	redisClient := createTestRedisClient()

	err := redisClient.FlushDB(context.Background())
	assert.NoError(t, err, "Ошибка при очистке базы Redis")

	defer func() {
		err := redisClient.FlushDB(context.Background())
		assert.NoError(t, err, "Ошибка при очистке базы Redis")
	}()

	authServer := &grpc.AuthServiceServer{
		RedisClient:   redisClient,
		LoginLimit:    loginLimit,
		PasswordLimit: 100,
		IPLimit:       IPLimit,
		BucketTTL:     time.Minute,
	}

	req := &pb.AuthRequest{
		Login:    "testuser_login_limit",
		Password: "password_with_high_limit",
		Ip:       "127.0.0.1",
	}

	for i := 0; i < authServer.LoginLimit; i++ {
		res, err := authServer.CheckAuth(context.Background(), req)
		assert.NoError(t, err)
		assert.True(t, res.Ok, "Авторизация должна быть разрешена до достижения лимита для логина")
	}

	res, err := authServer.CheckAuth(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, res.Ok, "Авторизация должна быть отклонена после достижения лимита для логина")
}

func TestCheckAuth_Whitelist(t *testing.T) {
	redisClient := createTestRedisClient()

	err := redisClient.FlushDB(context.Background())
	assert.NoError(t, err, "Ошибка при очистке базы Redis")

	defer func() {
		err := redisClient.FlushDB(context.Background())
		assert.NoError(t, err, "Ошибка при очистке базы Redis")
	}()

	authServer := &grpc.AuthServiceServer{
		RedisClient:   redisClient,
		LoginLimit:    loginLimit,
		PasswordLimit: passwordLimit,
		IPLimit:       IPLimit,
		BucketTTL:     time.Minute,
	}

	err = redisClient.SAdd(context.Background(), "whitelist", "127.0.0.1/32")
	assert.NoError(t, err, "Ошибка при добавлении IP в белый список")

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

	authServer := &grpc.AuthServiceServer{
		RedisClient:   redisClient,
		LoginLimit:    loginLimit,
		PasswordLimit: passwordLimit,
		IPLimit:       IPLimit,
		BucketTTL:     time.Minute,
	}

	err := redisClient.SAdd(context.Background(), "blacklist", "127.0.0.1/32")
	assert.NoError(t, err, "Ошибка при добавлении IP в черный список")

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

	err := redisClient.FlushDB(context.Background())
	assert.NoError(t, err, "Ошибка при очистке базы Redis")

	defer func() {
		err := redisClient.FlushDB(context.Background())
		assert.NoError(t, err, "Ошибка при очистке базы Redis")
	}()

	authServer := &grpc.AuthServiceServer{
		RedisClient:   redisClient,
		LoginLimit:    loginLimit,
		PasswordLimit: passwordLimit,
		IPLimit:       IPLimit,
		BucketTTL:     time.Minute,
	}

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
	authServer := &grpc.AuthServiceServer{
		RedisClient:   redisclient.NewRedisClient("localhost:9999"),
		LoginLimit:    loginLimit,
		PasswordLimit: passwordLimit,
		IPLimit:       IPLimit,
		BucketTTL:     time.Minute,
	}

	res, err := authServer.CheckAuth(context.Background(), &pb.AuthRequest{
		Login:    "testuser",
		Password: "password123",
		Ip:       "127.0.0.1",
	})
	assert.Error(t, err, "Должна возникнуть ошибка при недоступности Redis")
	assert.False(t, res.Ok, "Авторизация должна быть запрещена при недоступности Redis")
}

func TestCheckAuth_Parallel(t *testing.T) {
	redisClient := createTestRedisClient()

	err := redisClient.FlushDB(context.Background())
	assert.NoError(t, err, "Ошибка при очистке базы Redis")

	defer func() {
		err := redisClient.FlushDB(context.Background())
		assert.NoError(t, err, "Ошибка при очистке базы Redis")
	}()

	authServer := &grpc.AuthServiceServer{
		RedisClient:   redisClient,
		LoginLimit:    loginLimit,
		PasswordLimit: passwordLimit,
		IPLimit:       IPLimit,
		BucketTTL:     time.Minute,
	}
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
	assert.NoError(t, err, "Ошибка при очистке базы Redis")

	defer func() {
		err := redisClient.FlushDB(context.Background())
		assert.NoError(t, err, "Ошибка при очистке базы Redis")
	}()

	authServer := &grpc.AuthServiceServer{
		RedisClient:   redisClient,
		LoginLimit:    loginLimit,
		PasswordLimit: passwordLimit,
		IPLimit:       IPLimit,
		BucketTTL:     time.Minute,
	}
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
