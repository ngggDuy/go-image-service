package main

import (
	"context"
	"log"
	"net/http"

	"go-image-service/internal/config"
	"go-image-service/internal/metadata"
	"go-image-service/internal/pipeline"
	"go-image-service/internal/storage"
	"go-image-service/internal/transport"
	"go-image-service/internal/upload"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/client"
)

func main() {
	cfg := config.Load()

	// Temporal client (used to start the resize workflow).
	tc, err := client.Dial(client.Options{HostPort: cfg.TemporalAddress})
	if err != nil {
		log.Fatalf("dial temporal: %v", err)
	}
	defer tc.Close()

	// Database connection pool.
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("cannot reach postgres: %v", err)
	}

	// File storage on disk.
	store, err := storage.New("uploads")
	if err != nil {
		log.Fatal(err)
	}

	// Wire the upload service via temporal and the HTTP server.
	repo := metadata.New(pool)
	svc := upload.New(store, repo, pipeline.NewOrchestrator(tc, cfg.TaskQueue))
	server := transport.New(svc, store, repo)

	log.Printf("http server listening on %s", cfg.HTTPAddr)
	log.Fatal(http.ListenAndServe(cfg.HTTPAddr, server.Routes()))
}
