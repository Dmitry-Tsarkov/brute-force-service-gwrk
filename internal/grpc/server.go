package grpc

import (
	"log"
	"net"

	"google.golang.org/grpc"

	//nolint:depguard
	pb "github.com/Dmitry-Tsarkov/brute-force-service-grw/api"
)

func StartGRPCServer() {
	lis, err := net.Listen("tcp", "localhost:50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterAuthServiceServer(s, &AuthServiceServer{})

	log.Println("gRPC server running on port 50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
