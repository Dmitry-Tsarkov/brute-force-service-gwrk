package main

import (
	"context"
	"fmt"
	"log"
	"net"

	//nolint:depguard
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	//nolint:depguard
	pb "github.com/Dmitry-Tsarkov/brute-force-service-gwrk/api"
	//nolint:depguard
	"github.com/Dmitry-Tsarkov/brute-force-service-gwrk/internal/config"
	//nolint:depguard
	authgrpc "github.com/Dmitry-Tsarkov/brute-force-service-gwrk/internal/grpc"
	//nolint:depguard
	"github.com/Dmitry-Tsarkov/brute-force-service-gwrk/internal/redisclient"
)

var (
	login string
	ip    string
)

var RootCmd = &cobra.Command{
	Use:   "bruteforce-cli",
	Short: "CLI для управления brute-force сервисом",
}

func main() {
	cfg := config.LoadConfig()

	redisAddr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)
	rdb := redisclient.NewRedisClient(redisAddr)

	ctx := context.Background()
	if _, err := rdb.Get(ctx, "test"); err != nil {
		log.Fatalf("Не удалось подключиться к Redis: %v", err)
	}
	log.Println("Успешно подключено к Redis")

	listenAddr := fmt.Sprintf(":%s", cfg.GRPCPort)
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("Не удалось слушать порт: %v", err)
	}

	s := grpc.NewServer()
	authServer := &authgrpc.AuthServiceServer{
		RedisClient: rdb,
		Config:      cfg,
	}

	pb.RegisterAuthServiceServer(s, authServer)
	reflection.Register(s)

	log.Printf("gRPC сервер запущен на порту %s с лимитами LoginLimit=%d, PasswordLimit=%d, IPLimit=%d",
		cfg.GRPCPort, cfg.LoginLimit, cfg.PasswordLimit, cfg.IPLimit)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Ошибка при запуске gRPC сервера: %v", err)
	}
}

func init() {
	RootCmd.AddCommand(resetBucketCmd)
	RootCmd.AddCommand(addToBlacklistCmd)
	RootCmd.AddCommand(addToWhitelistCmd)
	RootCmd.AddCommand(removeFromWhitelistCmd)

	resetBucketCmd.Flags().StringVar(&login, "login", "", "Логин для сброса бакетов")
	resetBucketCmd.Flags().StringVar(&ip, "ip", "", "IP для сброса бакетов")
	addToBlacklistCmd.Flags().StringVar(&ip, "ip", "", "IP для добавления в черный список")
	addToWhitelistCmd.Flags().StringVar(&ip, "ip", "", "IP для добавления в белый список")
	removeFromWhitelistCmd.Flags().StringVar(&ip, "ip", "", "IP для удаления из белого списка")
}

var resetBucketCmd = &cobra.Command{
	Use:   "reset-bucket",
	Short: "Сбросить бакеты для логина и IP",
	Run: func(cmd *cobra.Command, _ []string) {
		handleResetBucket(cmd)
	},
}

var addToBlacklistCmd = &cobra.Command{
	Use:   "add-to-blacklist",
	Short: "Добавить IP в черный список",
	Run: func(cmd *cobra.Command, _ []string) {
		handleAddToBlacklist(cmd)
	},
}

var addToWhitelistCmd = &cobra.Command{
	Use:   "add-to-whitelist",
	Short: "Добавить IP в белый список",
	Run: func(cmd *cobra.Command, _ []string) {
		handleAddToWhitelist(cmd)
	},
}

var removeFromWhitelistCmd = &cobra.Command{
	Use:   "remove-from-whitelist",
	Short: "Удалить IP из белого списка",
	Run: func(cmd *cobra.Command, _ []string) {
		handleRemoveFromWhitelist(cmd)
	},
}

func handleResetBucket(cmd *cobra.Command) {
	conn, err := grpc.DialContext(
		cmd.Context(),
		"localhost:"+config.LoadConfig().GRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
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
}

func handleAddToBlacklist(cmd *cobra.Command) {
	conn, err := grpc.DialContext(
		cmd.Context(),
		"localhost:"+config.LoadConfig().GRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Printf("Ошибка подключения к серверу: %v\n", err)
		return
	}
	defer conn.Close()

	client := pb.NewAuthServiceClient(conn)
	_, err = client.AddToBlacklist(cmd.Context(), &pb.ListRequest{Ip: ip})
	if err != nil {
		fmt.Printf("Ошибка добавления IP в черный список: %v\n", err)
		return
	}
	fmt.Println("IP успешно добавлен в черный список.")
}

func handleAddToWhitelist(cmd *cobra.Command) {
	conn, err := grpc.DialContext(
		cmd.Context(),
		"localhost:"+config.LoadConfig().GRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
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
}

func handleRemoveFromWhitelist(cmd *cobra.Command) {
	conn, err := grpc.DialContext(
		cmd.Context(),
		"localhost:"+config.LoadConfig().GRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
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
}
