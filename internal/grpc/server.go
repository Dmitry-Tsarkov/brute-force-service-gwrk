package grpc

import (
	"log"
	"net"
	"os"
	"strconv"

	//nolint:depguard
	pb "github.com/Dmitry-Tsarkov/brute-force-service-gwrk/api"
	//nolint:depguard
	"github.com/Dmitry-Tsarkov/brute-force-service-gwrk/internal/redisclient"
	"google.golang.org/grpc"
)

func getLimits() (int, int) {
	loginLimitStr := os.Getenv("LOGIN_LIMIT")
	IPLimitStr := os.Getenv("IP_LIMIT")

	loginLimit, err := strconv.Atoi(loginLimitStr)
	if err != nil {
		log.Printf("Ошибка чтения LOGIN_LIMIT, используем значение по умолчанию 10")
		loginLimit = 10
	}

	IPLimit, err := strconv.Atoi(IPLimitStr)
	if err != nil {
		log.Printf("Ошибка чтения IP_LIMIT, используем значение по умолчанию 100")
		IPLimit = 100
	}

	return loginLimit, IPLimit
}

func StartGRPCServer() {
	lis, err := net.Listen("tcp", "localhost:50051")
	if err != nil {
		log.Fatalf("Не удалось начать слушать порт: %v", err)
	}

	redisClient := redisclient.NewRedisClient("localhost:6379")
	loginLimit, IPLimit := getLimits()

	authServer := &AuthServiceServer{
		RedisClient: redisClient,
		LoginLimit:  loginLimit,
		IPLimit:     IPLimit,
	}

	s := grpc.NewServer()
	pb.RegisterAuthServiceServer(s, authServer)

	log.Printf("gRPC сервер работает на порту 50051 с лимитами LoginLimit=%d и IPLimit=%d", loginLimit, IPLimit)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Не удалось запустить сервер: %v", err)
	}
}
