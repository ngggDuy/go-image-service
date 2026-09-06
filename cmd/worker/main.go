package main

import (
	"context"
	"log"

	"go-image-service/gen/imageprocess"
	"go-image-service/internal/config"
	"go-image-service/internal/grpcauth"
	"go-image-service/internal/metadata"
	"go-image-service/internal/pipeline"
	"go-image-service/internal/resizer"
	"go-image-service/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	cfg := config.Load()

	conn, err := grpcauth.Dial(context.Background(), cfg.ImageServiceAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

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

	c, err := client.Dial(client.Options{HostPort: cfg.TemporalAddress})
	if err != nil {
		log.Fatalf("dial temporal: %v", err)
	}
	defer c.Close()

	w := worker.New(c, cfg.TaskQueue, worker.Options{})
	w.RegisterWorkflow(pipeline.UploadWorkflow)
	w.RegisterActivity(&pipeline.Activities{
		Resizer: resizer.New(imageprocess.NewResizerClient(conn)),
		Objects: objects,
		Repo:    metadata.New(pool),
	})

	log.Printf("worker listening on task queue %q", cfg.TaskQueue)
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("worker stopped: %v", err)
	}
}
