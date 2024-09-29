package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	//nolint:depguard
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	//nolint:depguard
	pb "github.com/Dmitry-Tsarkov/brute-force-service-gwrk/api"
	//nolint:depguard
	authgrpc "github.com/Dmitry-Tsarkov/brute-force-service-gwrk/internal/grpc"
	//nolint:depguard
	"github.com/Dmitry-Tsarkov/brute-force-service-gwrk/internal/redisclient"
)

func main() {
	loginLimit := 10
	passwordLimit := 5
	IPLimit := 100
	bucketTTL := time.Minute

	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	if redisHost == "" || redisPort == "" {
		log.Fatal("Redis environment variables not set")
	}
	redisAddr := redisHost + ":" + redisPort

	rdb := redisclient.NewRedisClient(redisAddr)

	ctx := context.Background()

	if _, err := rdb.Get(ctx, "test"); err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}

	log.Println("Redis connected successfully")

	lis, err := net.Listen("tcp", "localhost:50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	authServer := &authgrpc.AuthServiceServer{
		RedisClient:   rdb,
		LoginLimit:    loginLimit,
		PasswordLimit: passwordLimit,
		IPLimit:       IPLimit,
		BucketTTL:     bucketTTL,
	}

	pb.RegisterAuthServiceServer(s, authServer)
	reflection.Register(s)

	log.Println("gRPC server running on port 50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

var (
	login string
	ip    string
)

var rootCmd = &cobra.Command{
	Use:   "bruteforce-cli",
	Short: "CLI для управления brute-force сервисом",
}

var resetBucketCmd = &cobra.Command{
	Use:   "reset-bucket",
	Short: "Сбросить бакеты для логина и IP",
	Run: func(cmd *cobra.Command, _ []string) {
		conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Printf("Ошибка подключения к серверу: %v\n", err)
			return
		}
		defer conn.Close()

		client := pb.NewAuthServiceClient(conn)
		_, err = client.ResetBucket(cmd.Context(), &pb.ResetRequest{Login: login, Ip: ip})
		if err != nil {
			fmt.Printf("Ошибка сброса бакетов: %v\n", err)
			return
		}
		fmt.Println("Бакеты успешно сброшены.")
	},
}

var addToBlacklistCmd = &cobra.Command{
	Use:   "add-to-blacklist",
	Short: "Добавить IP в чёрный список",
	Run: func(cmd *cobra.Command, _ []string) {
		conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Printf("Ошибка подключения к серверу: %v\n", err)
			return
		}
		defer conn.Close()

		client := pb.NewAuthServiceClient(conn)
		_, err = client.AddToBlacklist(cmd.Context(), &pb.ListRequest{Ip: ip})
		if err != nil {
			fmt.Printf("Ошибка добавления IP в чёрный список: %v\n", err)
			return
		}
		fmt.Println("IP успешно добавлен в чёрный список.")
	},
}

var removeFromBlacklistCmd = &cobra.Command{
	Use:   "remove-from-blacklist",
	Short: "Удалить IP из чёрного списка",
	Run: func(cmd *cobra.Command, _ []string) {
		conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Printf("Ошибка подключения к серверу: %v\n", err)
			return
		}
		defer conn.Close()

		client := pb.NewAuthServiceClient(conn)
		_, err = client.RemoveFromBlacklist(cmd.Context(), &pb.ListRequest{Ip: ip})
		if err != nil {
			fmt.Printf("Ошибка удаления IP из чёрного списка: %v\n", err)
			return
		}
		fmt.Println("IP успешно удалён из чёрного списка.")
	},
}

var addToWhitelistCmd = &cobra.Command{
	Use:   "add-to-whitelist",
	Short: "Добавить IP в белый список",
	Run: func(cmd *cobra.Command, _ []string) {
		conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Printf("Ошибка подключения к серверу: %v\n", err)
			return
		}
		defer conn.Close()

		client := pb.NewAuthServiceClient(conn)
		_, err = client.AddToWhitelist(cmd.Context(), &pb.ListRequest{Ip: ip})
		if err != nil {
			fmt.Printf("Ошибка добавления IP в белый список: %v\n", err)
			return
		}
		fmt.Println("IP успешно добавлен в белый список.")
	},
}

var removeFromWhitelistCmd = &cobra.Command{
	Use:   "remove-from-whitelist",
	Short: "Удалить IP из белого списка",
	Run: func(cmd *cobra.Command, _ []string) {
		conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Printf("Ошибка подключения к серверу: %v\n", err)
			return
		}
		defer conn.Close()

		client := pb.NewAuthServiceClient(conn)
		_, err = client.RemoveFromWhitelist(cmd.Context(), &pb.ListRequest{Ip: ip})
		if err != nil {
			fmt.Printf("Ошибка удаления IP из белого списка: %v\n", err)
			return
		}
		fmt.Println("IP успешно удалён из белого списка.")
	},
}

func init() {
	rootCmd.AddCommand(resetBucketCmd)
	rootCmd.AddCommand(addToBlacklistCmd)
	rootCmd.AddCommand(removeFromBlacklistCmd)
	rootCmd.AddCommand(addToWhitelistCmd)
	rootCmd.AddCommand(removeFromWhitelistCmd)

	resetBucketCmd.Flags().StringVar(&login, "login", "", "Логин для сброса бакетов")
	resetBucketCmd.Flags().StringVar(&ip, "ip", "", "IP для сброса бакетов")

	addToBlacklistCmd.Flags().StringVar(&ip, "ip", "", "IP для добавления в чёрный список")
	removeFromBlacklistCmd.Flags().StringVar(&ip, "ip", "", "IP для удаления из чёрного списка")

	addToWhitelistCmd.Flags().StringVar(&ip, "ip", "", "IP для добавления в белый список")
	removeFromWhitelistCmd.Flags().StringVar(&ip, "ip", "", "IP для удаления из белого списка")
}

func Execute() error {
	return rootCmd.Execute()
}
