package main

import (
	"context"
	"log"

	"go-image-service/gen/imageprocess"
	"go-image-service/internal/config"
	"go-image-service/internal/metadata"
	"go-image-service/internal/pipeline"
	"go-image-service/internal/resizer"
	"go-image-service/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.Load()

	// gRPC connection to the image processing service.
	conn, err := grpc.NewClient(cfg.ImageServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Database + file storage (activities write results + status here).
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	store, err := storage.New("uploads")
	if err != nil {
		log.Fatal(err)
	}

	// Dial into the Temporal client + worker.
	c, err := client.Dial(client.Options{HostPort: cfg.TemporalAddress})
	if err != nil {
		log.Fatalf("dial temporal: %v", err)
	}
	defer c.Close()

	// Create worker and register workflows and activities
	w := worker.New(c, cfg.TaskQueue, worker.Options{})
	w.RegisterWorkflow(pipeline.UploadWorkflow)
	w.RegisterActivity(&pipeline.Activities{
		Resizer: resizer.New(imageprocess.NewResizerClient(conn)),
		Store:   store,
		Repo:    metadata.New(pool),
	})

	log.Printf("worker listening on task queue %q", cfg.TaskQueue)
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("worker stopped: %v", err)
	}
}
