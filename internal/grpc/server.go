package grpc

import (
	"log"
	"net"
	"os"

	//nolint:depguard
	pb "github.com/Dmitry-Tsarkov/brute-force-service-gwrk/api"
	//nolint:depguard
	"github.com/Dmitry-Tsarkov/brute-force-service-gwrk/internal/config"
	//nolint:depguard
	"github.com/Dmitry-Tsarkov/brute-force-service-gwrk/internal/redisclient"
	"google.golang.org/grpc"
)

const (
	defaultGRPCPort  = "50051"
	defaultRedisHost = "redis"
	defaultRedisPort = "6379"
)

func StartGRPCServer() {
	cfg := config.LoadConfig()

	redisHost := cfg.RedisHost
	if redisHost == "" {
		redisHost = defaultRedisHost
	}

	redisPort := cfg.RedisPort
	if redisPort == "" {
		redisPort = defaultRedisPort
	}

	redisAddr := redisHost + ":" + redisPort

	redisClient := redisclient.NewRedisClient(redisAddr)

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = defaultGRPCPort
	}

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Не удалось начать слушать порт %s: %v", grpcPort, err)
	}

	authServer := &AuthServiceServer{
		RedisClient: redisClient,
		Config:      cfg,
	}

	s := grpc.NewServer()
	pb.RegisterAuthServiceServer(s, authServer)

	log.Printf("gRPC сервер работает на порту %s с лимитами: LoginLimit=%d, PasswordLimit=%d и IPLimit=%d",
		grpcPort, cfg.LoginLimit, cfg.PasswordLimit, cfg.IPLimit)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Не удалось запустить сервер: %v", err)
	}
}
