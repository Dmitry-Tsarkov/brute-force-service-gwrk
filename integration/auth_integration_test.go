//go:build integration
// +build integration

package integration_test

import (
	"context"
	"log"
	"net"
	"testing"
	"time"

	//nolint:depguard
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	//nolint:depguard
	pb "github.com/Dmitry-Tsarkov/brute-force-service-grw/api"
	// nolint:depguard
	authgrpc "github.com/Dmitry-Tsarkov/brute-force-service-grw/internal/grpc"
	// nolint:depguard
	"github.com/Dmitry-Tsarkov/brute-force-service-grw/internal/redisclient"
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
	authServer := &authgrpc.AuthServiceServer{RedisClient: redisClient}
	pb.RegisterAuthServiceServer(grpcServer, authServer)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	return grpcServer, lis, lis.Addr().String()
}

func TestCheckAuth_Integration(t *testing.T) {
	ctx := context.Background()
	redisClient := createTestRedisClient()
	defer redisClient.FlushDB(ctx)
	//test.
	grpcServer, _, address := startGRPCServer(redisClient)
	defer grpcServer.Stop()

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewAuthServiceClient(conn)

	req := &pb.AuthRequest{
		Login:    "testuser",
		Password: "password123",
		Ip:       "127.0.0.1",
	}
	resp, err := client.CheckAuth(ctx, req)
	assert.NoError(t, err)
	assert.True(t, resp.Ok, "Авторизация должна быть разрешена")

	for i := 0; i < 11; i++ {
		_, _ = client.CheckAuth(ctx, req)
	}

	resp, err = client.CheckAuth(ctx, req)
	assert.NoError(t, err)
	assert.False(t, resp.Ok, "Авторизация должна быть отклонена при превышении лимита")
}

func TestResetBucket_Integration(t *testing.T) {
	ctx := context.Background()
	redisClient := createTestRedisClient()
	defer redisClient.FlushDB(ctx)

	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	authServer := &authgrpc.AuthServiceServer{RedisClient: redisClient}
	pb.RegisterAuthServiceServer(grpcServer, authServer)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	defer grpcServer.Stop()

	address := lis.Addr().String()
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewAuthServiceClient(conn)

	req := &pb.AuthRequest{
		Login:    "testuser",
		Password: "password123",
		Ip:       "127.0.0.1",
	}
	for i := 0; i < 11; i++ {
		_, _ = client.CheckAuth(ctx, req)
	}

	resetReq := &pb.ResetRequest{
		Login: "testuser",
		Ip:    "127.0.0.1",
	}
	resetResp, err := client.ResetBucket(ctx, resetReq)
	assert.NoError(t, err)
	assert.True(t, resetResp.Status, "Бакеты должны быть успешно сброшены")
	resp, err := client.CheckAuth(ctx, req)
	assert.NoError(t, err)
	assert.True(t, resp.Ok, "Авторизация должна быть разрешена после сброса")
}

func TestWhitelist_Integration(t *testing.T) {
	ctx := context.Background()
	redisClient := createTestRedisClient()
	defer redisClient.FlushDB(ctx)

	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("gRPC сервер запущен на порту %s", lis.Addr().String())

	grpcServer := grpc.NewServer()
	authServer := &authgrpc.AuthServiceServer{RedisClient: redisClient}
	pb.RegisterAuthServiceServer(grpcServer, authServer)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	defer grpcServer.Stop()

	address := lis.Addr().String()
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewAuthServiceClient(conn)
	err = redisClient.Set(ctx, "whitelist:127.0.0.1", true, time.Hour)
	assert.NoError(t, err)
	req := &pb.AuthRequest{
		Login:    "testuser",
		Password: "password123",
		Ip:       "127.0.0.1",
	}
	resp, err := client.CheckAuth(ctx, req)
	assert.NoError(t, err)
	assert.True(t, resp.Ok, "Авторизация должна быть разрешена для IP в белом списке")
}

func TestBlacklist_Integration(t *testing.T) {
	ctx := context.Background()
	redisClient := createTestRedisClient()
	defer redisClient.FlushDB(ctx)
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("gRPC сервер запущен на порту %s", lis.Addr().String())

	grpcServer := grpc.NewServer()
	authServer := &authgrpc.AuthServiceServer{RedisClient: redisClient}
	pb.RegisterAuthServiceServer(grpcServer, authServer)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	defer grpcServer.Stop()
	address := lis.Addr().String()
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewAuthServiceClient(conn)
	err = redisClient.Set(ctx, "blacklist:127.0.0.1", "true", time.Hour)
	assert.NoError(t, err)

	req := &pb.AuthRequest{
		Login:    "testuser",
		Password: "password123",
		Ip:       "127.0.0.1",
	}
	resp, err := client.CheckAuth(ctx, req)
	assert.NoError(t, err)
	assert.False(t, resp.Ok, "Авторизация должна быть запрещена для IP в черном списке")
}
