package grpc_test

import (
	"context"
	"testing"
	"time"

	//nolint:depguard
	"github.com/stretchr/testify/assert"
	//nolint:depguard
	pb "github.com/Dmitry-Tsarkov/brute-force-service-gwrk/api"
	//nolint:depguard
	"github.com/Dmitry-Tsarkov/brute-force-service-gwrk/internal/grpc"
	//nolint:depguard
	"github.com/Dmitry-Tsarkov/brute-force-service-gwrk/internal/config"
	//nolint:depguard
	"github.com/Dmitry-Tsarkov/brute-force-service-gwrk/internal/redisclient"
)

func createTestRedisClient() *redisclient.Client {
	return redisclient.NewRedisClient("localhost:6379")
}

func createTestConfig() *config.Config {
	return &config.Config{
		LoginLimit:    10,
		PasswordLimit: 5,
		IPLimit:       100,
		BucketTTL:     time.Minute,
	}
}

func TestCheckAuth_WithinLimit(t *testing.T) {
	redisClient := createTestRedisClient()

	err := redisClient.FlushDB(context.Background())
	assert.NoError(t, err, "Ошибка при очистке базы Redis")

	defer func() {
		err := redisClient.FlushDB(context.Background())
		assert.NoError(t, err, "Ошибка при очистке базы Redis")
	}()

	cfg := createTestConfig()

	authServer := &grpc.AuthServiceServer{
		RedisClient: redisClient,
		Config:      cfg,
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

	cfg := createTestConfig()

	authServer := &grpc.AuthServiceServer{
		RedisClient: redisClient,
		Config:      cfg,
	}

	req := &pb.AuthRequest{
		Login:    "testuser",
		Password: "password_limit_test",
		Ip:       "127.0.0.1",
	}

	for i := 0; i < authServer.Config.PasswordLimit; i++ {
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

	cfg := &config.Config{
		LoginLimit:    10,
		PasswordLimit: 100,
		IPLimit:       100,
		BucketTTL:     time.Minute,
	}

	authServer := &grpc.AuthServiceServer{
		RedisClient: redisClient,
		Config:      cfg,
	}

	req := &pb.AuthRequest{
		Login:    "testuser_login_limit",
		Password: "password_with_high_limit",
		Ip:       "127.0.0.1",
	}

	for i := 0; i < authServer.Config.LoginLimit; i++ {
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

	cfg := createTestConfig()

	authServer := &grpc.AuthServiceServer{
		RedisClient: redisClient,
		Config:      cfg,
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
		err := redisClient.FlushDB(context.Background())
		assert.NoError(t, err, "Ошибка при очистке базы Redis")
	}()

	cfg := createTestConfig()

	authServer := &grpc.AuthServiceServer{
		RedisClient: redisClient,
		Config:      cfg,
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

	cfg := createTestConfig()

	authServer := &grpc.AuthServiceServer{
		RedisClient: redisClient,
		Config:      cfg,
	}

	for i := 0; i < 3; i++ {
		_, _ = authServer.CheckAuth(context.Background(), &pb.AuthRequest{
			Login:    "testuser",
			Password: "password123",
			Ip:       "127.0.0.1",
		})
	}

	_, err = authServer.ResetBucket(context.Background(), &pb.ResetRequest{
		Login:    "testuser",
		Ip:       "127.0.0.1",
		Password: "password123",
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

func TestCheckAuth_WhitelistWithSubnet(t *testing.T) {
	redisClient := createTestRedisClient()

	err := redisClient.FlushDB(context.Background())
	assert.NoError(t, err, "Ошибка при очистке базы Redis")

	defer func() {
		err := redisClient.FlushDB(context.Background())
		assert.NoError(t, err, "Ошибка при очистке базы Redis")
	}()

	cfg := createTestConfig()

	authServer := &grpc.AuthServiceServer{
		RedisClient: redisClient,
		Config:      cfg,
	}

	err = redisClient.SAdd(context.Background(), "whitelist", "192.168.1.0/24")
	assert.NoError(t, err, "Ошибка при добавлении подсети в белый список")

	req := &pb.AuthRequest{
		Login:    "testuser",
		Password: "password123",
		Ip:       "192.168.1.5",
	}
	res, err := authServer.CheckAuth(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, res.Ok, "Авторизация должна быть разрешена для IP в подсети белого списка")
}

func TestCheckAuth_BlacklistWithSubnet(t *testing.T) {
	redisClient := createTestRedisClient()

	err := redisClient.FlushDB(context.Background())
	assert.NoError(t, err, "Ошибка при очистке базы Redis")

	defer func() {
		err := redisClient.FlushDB(context.Background())
		assert.NoError(t, err, "Ошибка при очистке базы Redis")
	}()

	cfg := createTestConfig()

	authServer := &grpc.AuthServiceServer{
		RedisClient: redisClient,
		Config:      cfg,
	}

	err = redisClient.SAdd(context.Background(), "blacklist", "192.168.1.0/24")
	assert.NoError(t, err, "Ошибка при добавлении подсети в черный список")

	req := &pb.AuthRequest{
		Login:    "testuser",
		Password: "password123",
		Ip:       "192.168.1.5",
	}
	res, err := authServer.CheckAuth(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, res.Ok, "Авторизация должна быть запрещена для IP в подсети черного списка")
}
