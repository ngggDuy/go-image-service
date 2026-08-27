package main

import (
	"context"
	"log"
	"net/http"

	"go-image-service/internal/authclient"
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

	tc, err := client.Dial(client.Options{HostPort: cfg.TemporalAddress})
	if err != nil {
		log.Fatalf("dial temporal: %v", err)
	}
	defer tc.Close()

	authClient, err := authclient.Dial(cfg.AuthServiceAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer authClient.Close()

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("cannot reach postgres: %v", err)
	}

	// Object storage (MinIO). Ensure the bucket exists at startup.
	objects, err := storage.NewMinioStore(
		cfg.MinioEndpoint, cfg.MinioPublicEndpoint,
		cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucket, false,
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := objects.EnsureBucket(context.Background()); err != nil {
		log.Fatalf("ensure bucket: %v", err)
	}

	repo := metadata.New(pool)
	uploadSvc := upload.New(repo, objects, pipeline.NewOrchestrator(tc, cfg.TaskQueue))
	server := transport.New(authClient, uploadSvc, repo, objects)

	log.Printf("http server listening on %s", cfg.HTTPAddr)
	log.Fatal(http.ListenAndServe(cfg.HTTPAddr, server.Routes()))
}
