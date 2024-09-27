package grpc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	//nolint:depguard
	"github.com/go-redis/redis/v8"
	//nolint:depguard
	pb "github.com/Dmitry-Tsarkov/brute-force-service-grw/api"
	//nolint:depguard
	"github.com/Dmitry-Tsarkov/brute-force-service-grw/internal/redisclient"
)

type AuthServiceServer struct {
	RedisClient redisclient.RedisClient
	pb.UnimplementedAuthServiceServer
}

const (
	loginLimit = 10
	ipLimit    = 100
	bucketTTL  = 60 * time.Second
)

func (s *AuthServiceServer) CheckAuth(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	if req.Login == "" || req.Password == "" || req.Ip == "" {
		log.Printf("Некорректные данные: login, password или ip отсутствуют")
		return &pb.AuthResponse{Ok: false, Error: "Некорректные данные"}, nil
	}

	whitelistKey := fmt.Sprintf("whitelist:%s", req.Ip)
	isWhitelisted, err := s.RedisClient.Get(ctx, whitelistKey)
	if err != nil && !errors.Is(err, redis.Nil) {
		log.Printf("Ошибка Redis при проверке белого списка: %v", err)
		return &pb.AuthResponse{Ok: false, Error: "Ошибка проверки белого списка"}, err
	}
	if isWhitelisted == "true" {
		log.Printf("IP %s находится в белом списке, авторизация разрешена", req.Ip)
		return &pb.AuthResponse{Ok: true}, nil
	}

	blacklistKey := fmt.Sprintf("blacklist:%s", req.Ip)
	isBlacklisted, err := s.RedisClient.Get(ctx, blacklistKey)
	if err != nil && !errors.Is(err, redis.Nil) {
		log.Printf("Ошибка Redis при проверке черного списка: %v", err)
		return &pb.AuthResponse{Ok: false, Error: "Ошибка проверки черного списка"}, err
	}
	if isBlacklisted == "true" {
		log.Printf("IP %s находится в черном списке, авторизация запрещена", req.Ip)
		return &pb.AuthResponse{Ok: false, Error: "IP находится в черном списке"}, nil
	}

	loginKey := fmt.Sprintf("login:%s", req.Login)
	if allowed, err := s.checkLimit(ctx, loginKey, loginLimit, bucketTTL); err != nil {
		log.Printf("Ошибка Redis при проверке лимита для логина %s: %v", req.Login, err)
		return &pb.AuthResponse{Ok: false, Error: "Ошибка проверки лимита для логина"}, err
	} else if !allowed {
		log.Printf("Превышен лимит для логина: %s", req.Login)
		return &pb.AuthResponse{Ok: false, Error: fmt.Sprintf("Превышен лимит для логина: %s", req.Login)}, nil
	}

	ipKey := fmt.Sprintf("ip:%s", req.Ip)
	if allowed, err := s.checkLimit(ctx, ipKey, ipLimit, bucketTTL); err != nil {
		log.Printf("Ошибка Redis при проверке лимита для IP %s: %v", req.Ip, err)
		return &pb.AuthResponse{Ok: false, Error: "Ошибка проверки лимита для IP"}, err
	} else if !allowed {
		log.Printf("Превышен лимит для IP: %s", req.Ip)
		return &pb.AuthResponse{Ok: false, Error: fmt.Sprintf("Превышен лимит для IP: %s", req.Ip)}, nil
	}

	log.Printf("Авторизация разрешена для логина: %s, IP: %s", req.Login, req.Ip)
	return &pb.AuthResponse{Ok: true}, nil
}

func (s *AuthServiceServer) checkLimit(ctx context.Context, key string, limit int, ttl time.Duration) (bool, error) {
	attempts, err := s.RedisClient.Incr(ctx, key)
	if err != nil {
		log.Printf("Ошибка Redis при инкременте для ключа %s: %v", key, err)
		return false, err
	}
	log.Printf("Попытки для ключа %s: %d", key, attempts)
	if err := s.RedisClient.SetTTL(ctx, key, ttl); err != nil {
		log.Printf("Ошибка установки TTL для ключа %s: %v", key, err)
	} else {
		log.Printf("TTL для ключа %s установлен на %v", key, ttl)
	}
	if attempts > int64(limit) {
		log.Printf("Превышен лимит для ключа %s: %d/%d", key, attempts, limit)
		return false, nil
	}

	return true, nil
}

func (s *AuthServiceServer) ResetBucket(ctx context.Context, req *pb.ResetRequest) (*pb.ResetResponse, error) {
	loginKey := fmt.Sprintf("login:%s", req.Login)
	ipKey := fmt.Sprintf("ip:%s", req.Ip)

	err := s.RedisClient.Del(ctx, loginKey, ipKey)
	if err != nil {
		log.Printf("Ошибка сброса бакетов: %v", err)
		return &pb.ResetResponse{Status: false}, err
	}

	log.Printf("Бакеты успешно сброшены для логина: %s, IP: %s", req.Login, req.Ip)
	return &pb.ResetResponse{Status: true}, nil
}

func (s *AuthServiceServer) AddToBlacklist(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	blacklistKey := "blacklist:" + req.Ip

	err := s.RedisClient.Set(ctx, blacklistKey, true, time.Hour*24)
	if err != nil {
		log.Printf("Error adding IP to blacklist: %v", err)
		return &pb.ListResponse{Status: false}, err
	}

	return &pb.ListResponse{Status: true}, nil
}

func (s *AuthServiceServer) RemoveFromBlacklist(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	blacklistKey := "blacklist:" + req.Ip

	err := s.RedisClient.Del(ctx, blacklistKey)
	if err != nil {
		log.Printf("Error removing IP from blacklist: %v", err)
		return &pb.ListResponse{Status: false}, err
	}
	return &pb.ListResponse{Status: true}, nil
}

func (s *AuthServiceServer) AddToWhitelist(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	whitelistKey := "whitelist:" + req.Ip
	err := s.RedisClient.Set(ctx, whitelistKey, true, time.Hour*24)
	if err != nil {
		log.Printf("Error adding IP to whitelist: %v", err)
		return &pb.ListResponse{Status: false}, err
	}
	return &pb.ListResponse{Status: true}, nil
}

func (s *AuthServiceServer) RemoveFromWhitelist(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	whitelistKey := "whitelist:" + req.Ip
	err := s.RedisClient.Del(ctx, whitelistKey)
	if err != nil {
		log.Printf("Error removing IP from whitelist: %v", err)
		return &pb.ListResponse{Status: false}, err
	}
	return &pb.ListResponse{Status: true}, nil
}
