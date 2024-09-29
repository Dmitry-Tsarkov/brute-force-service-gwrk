//go:build integration
// +build integration

package integration_test

import (
	"context"
	"log"
	"net"
	"testing"

	//nolint:depguard
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	//nolint:depguard
	pb "github.com/Dmitry-Tsarkov/brute-force-service-gwrk/api"
	//nolint:depguard
	authgrpc "github.com/Dmitry-Tsarkov/brute-force-service-gwrk/internal/grpc"
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

func startGRPCServer(redisClient *redisclient.Client) (*grpc.Server, net.Listener, string) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	authServer := &authgrpc.AuthServiceServer{
		RedisClient:   redisClient,
		LoginLimit:    loginLimit,
		PasswordLimit: passwordLimit,
		IPLimit:       IPLimit,
	}
	pb.RegisterAuthServiceServer(grpcServer, authServer)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	return grpcServer, lis, lis.Addr().String()
}

func TestResetBucket_Integration(t *testing.T) {
	ctx := context.Background()
	redisClient := createTestRedisClient()
	defer redisClient.FlushDB(ctx)

	grpcServer, _, address := startGRPCServer(redisClient)
	defer grpcServer.Stop()

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewAuthServiceClient(conn)

	req := &pb.AuthRequest{
		Login:    "testuser_reset_bucket",
		Password: "password123",
		Ip:       "127.0.0.1",
	}

	for i := 0; i < 5; i++ {
		_, _ = client.CheckAuth(ctx, req)
	}

	resetReq := &pb.ResetRequest{
		Login: "testuser_reset_bucket",
		Ip:    "127.0.0.1",
	}
	resetResp, err := client.ResetBucket(ctx, resetReq)
	assert.NoError(t, err)
	assert.True(t, resetResp.Status, "Бакеты должны быть успешно сброшены")

	res, err := client.CheckAuth(ctx, req)
	assert.NoError(t, err)
	assert.True(t, res.Ok, "Авторизация должна быть разрешена после сброса")
}

func TestWhitelist_Integration(t *testing.T) {
	ctx := context.Background()
	redisClient := createTestRedisClient()
	defer redisClient.FlushDB(ctx)

	grpcServer, _, address := startGRPCServer(redisClient)
	defer grpcServer.Stop()

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewAuthServiceClient(conn)

	err = redisClient.SAdd(ctx, "whitelist", "127.0.0.1/32")
	assert.NoError(t, err)

	req := &pb.AuthRequest{
		Login:    "testuser",
		Password: "password123",
		Ip:       "127.0.0.1",
	}
	res, err := client.CheckAuth(ctx, req)
	assert.NoError(t, err)
	assert.True(t, res.Ok, "Авторизация должна быть разрешена для IP в белом списке")
}

func TestBlacklist_Integration(t *testing.T) {
	ctx := context.Background()
	redisClient := createTestRedisClient()
	defer redisClient.FlushDB(ctx)

	grpcServer, _, address := startGRPCServer(redisClient)
	defer grpcServer.Stop()

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewAuthServiceClient(conn)

	err = redisClient.SAdd(ctx, "blacklist", "127.0.0.1/32")
	assert.NoError(t, err)

	req := &pb.AuthRequest{
		Login:    "testuser",
		Password: "password123",
		Ip:       "127.0.0.1",
	}
	res, err := client.CheckAuth(ctx, req)
	assert.NoError(t, err)
	assert.False(t, res.Ok, "Авторизация должна быть запрещена для IP в черном списке")
}
