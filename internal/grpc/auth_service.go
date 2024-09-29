package grpc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	//nolint:depguard
	pb "github.com/Dmitry-Tsarkov/brute-force-service-gwrk/api"
	//nolint:depguard
	"github.com/Dmitry-Tsarkov/brute-force-service-gwrk/internal/redisclient"
)

type AuthServiceServer struct {
	RedisClient   redisclient.RedisClient
	LoginLimit    int
	PasswordLimit int
	IPLimit       int
	BucketTTL     time.Duration
	pb.UnimplementedAuthServiceServer
}

func (s *AuthServiceServer) CheckAuth(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	if req.Login == "" || req.Password == "" || req.Ip == "" {
		log.Printf("Некорректные данные: login, password или ip отсутствуют")
		return &pb.AuthResponse{Ok: false, Error: "Некорректные данные"}, nil
	}

	passwordKey := fmt.Sprintf("password:%s", req.Password)
	if allowed, err := s.checkLimit(ctx, passwordKey, s.PasswordLimit, s.BucketTTL); err != nil {
		log.Printf("Ошибка Redis при проверке лимита для пароля %s: %v", req.Password, err)
		return &pb.AuthResponse{Ok: false, Error: "Ошибка проверки лимита для пароля"}, err
	} else if !allowed {
		log.Printf("Превышен лимит для пароля: %s", req.Password)
		return &pb.AuthResponse{Ok: false, Error: fmt.Sprintf("Превышен лимит для пароля: %s", req.Password)}, nil
	}

	loginKey := fmt.Sprintf("login:%s", req.Login)
	if allowed, err := s.checkLimit(ctx, loginKey, s.LoginLimit, s.BucketTTL); err != nil {
		log.Printf("Ошибка Redis при проверке лимита для логина %s: %v", req.Login, err)
		return &pb.AuthResponse{Ok: false, Error: "Ошибка проверки лимита для логина"}, err
	} else if !allowed {
		log.Printf("Превышен лимит для логина: %s", req.Login)
		return &pb.AuthResponse{Ok: false, Error: fmt.Sprintf("Превышен лимит для логина: %s", req.Login)}, nil
	}

	ipKey := fmt.Sprintf("ip:%s", req.Ip)
	if allowed, err := s.checkLimit(ctx, ipKey, s.IPLimit, s.BucketTTL); err != nil {
		log.Printf("Ошибка Redis при проверке лимита для IP %s: %v", req.Ip, err)
		return &pb.AuthResponse{Ok: false, Error: "Ошибка проверки лимита для IP"}, err
	} else if !allowed {
		log.Printf("Превышен лимит для IP: %s", req.Ip)
		return &pb.AuthResponse{Ok: false, Error: fmt.Sprintf("Превышен лимит для IP: %s", req.Ip)}, nil
	}

	whitelistMembers, err := s.RedisClient.SMembers(ctx, "whitelist")
	if err != nil {
		log.Printf("Ошибка Redis при проверке белого списка: %v", err)
		return &pb.AuthResponse{Ok: false, Error: "Ошибка Redis при проверке белого списка"}, err
	}
	for _, subnet := range whitelistMembers {
		_, ipNet, err := net.ParseCIDR(subnet)
		if err != nil {
			log.Printf("Ошибка при разборе подсети %s: %v", subnet, err)
			continue
		}
		if ipNet.Contains(net.ParseIP(req.Ip)) {
			log.Printf("IP %s находится в белом списке, авторизация разрешена", req.Ip)
			return &pb.AuthResponse{Ok: true}, nil
		}
	}

	blacklistMembers, err := s.RedisClient.SMembers(ctx, "blacklist")
	if err != nil {
		log.Printf("Ошибка Redis при проверке черного списка: %v", err)
		return &pb.AuthResponse{Ok: false, Error: "Ошибка Redis при проверке черного списка"}, err
	}
	for _, subnet := range blacklistMembers {
		_, ipNet, err := net.ParseCIDR(subnet)
		if err != nil {
			log.Printf("Ошибка при разборе подсети %s: %v", subnet, err)
			continue
		}
		if ipNet.Contains(net.ParseIP(req.Ip)) {
			log.Printf("IP %s находится в черном списке, авторизация запрещена", req.Ip)
			return &pb.AuthResponse{Ok: false, Error: "IP находится в черном списке"}, nil
		}
	}

	log.Printf("Авторизация разрешена для логина: %s, IP: %s", req.Login, req.Ip)
	return &pb.AuthResponse{Ok: true}, nil
}

func (s *AuthServiceServer) checkLimit(ctx context.Context, key string, limit int, ttl time.Duration) (bool, error) {
	log.Printf("Проверка лимита для ключа: %s, лимит: %d, TTL: %v", key, limit, ttl)
	attempts, err := s.RedisClient.Incr(ctx, key)
	if err != nil {
		log.Printf("Ошибка Redis при инкременте для ключа %s: %v", key, err)
		return false, err
	}

	if attempts == 1 {
		log.Printf("Устанавливаем TTL для ключа %s: %v", key, ttl)
		err = s.RedisClient.SetTTL(ctx, key, ttl)
		if err != nil {
			log.Printf("Ошибка установки TTL для ключа %s: %v", key, err)
			return false, err
		}
	}
	log.Printf("Текущие попытки для ключа %s: %d/%d", key, attempts, limit)
	if attempts > int64(limit) {
		log.Printf("Превышен лимит для ключа %s: %d/%d", key, attempts, limit)
		return false, nil
	}

	return true, nil
}

func (s *AuthServiceServer) AddToBlacklist(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	_, _, err := net.ParseCIDR(req.Ip)
	if err != nil {
		log.Printf("Ошибка при разборе подсети %s: %v", req.Ip, err)
		return &pb.ListResponse{Status: false}, fmt.Errorf("некорректный IP или подсеть")
	}

	err = s.RedisClient.SAdd(ctx, "blacklist", req.Ip)
	if err != nil {
		log.Printf("Ошибка добавления IP в черный список: %v", err)
		return &pb.ListResponse{Status: false}, err
	}

	log.Printf("IP %s добавлен в черный список", req.Ip)
	return &pb.ListResponse{Status: true}, nil
}

func (s *AuthServiceServer) AddToWhitelist(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	_, _, err := net.ParseCIDR(req.Ip)
	if err != nil {
		log.Printf("Ошибка: неверный формат подсети %s", req.Ip)
		return &pb.ListResponse{Status: false}, errors.New("неверный формат подсети")
	}

	whitelistKey := "whitelist"
	err = s.RedisClient.SAdd(ctx, whitelistKey, req.Ip)
	if err != nil {
		log.Printf("Ошибка при добавлении подсети %s в белый список: %v", req.Ip, err)
		return &pb.ListResponse{Status: false}, err
	}

	return &pb.ListResponse{Status: true}, nil
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
