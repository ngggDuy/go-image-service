package main

import (
	"context"
	"log"
	"net"

	"go-image-service/gen/authpb"
	"go-image-service/internal/auth"
	"go-image-service/internal/authgrpc"
	"go-image-service/internal/config"
	"go-image-service/internal/user"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load()

	// This service owns the users table and the JWT secret.
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("cannot reach postgres: %v", err)
	}

	svc := auth.NewService(user.New(pool), []byte(cfg.JWTSecret))

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	authpb.RegisterAuthServiceServer(s, authgrpc.NewServer(svc))

	log.Println("auth service listening on :50052")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
