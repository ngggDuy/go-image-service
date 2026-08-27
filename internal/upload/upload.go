package upload

import (
	"context"
	"fmt"
	"time"

	"go-image-service/internal/id"
)

const presignTTL = 15 * time.Minute

// Repository persists metadata about an upload.
type Repository interface {
	Create(ctx context.Context, id, filename, contentType, status, userID string) error
}

// Presigner issues presigned URLs for direct client<->object-store transfers.
type Presigner interface {
	Presign(ctx context.Context, key, method string, ttl time.Duration) (string, error)
}

// Pipeline starts and signals the async resize workflow (Temporal in production).
type Pipeline interface {
	Start(ctx context.Context, id, contentType, filename string) error
	Signal(ctx context.Context, id string) error
}

// Service negotiates presigned uploads and forwards completion signals. It never
// handles image bytes — the client PUTs them straight to the object store.
type Service struct {
	repo    Repository
	objects Presigner
	pipe    Pipeline
}

func New(repo Repository, objects Presigner, pipe Pipeline) *Service {
	return &Service{repo: repo, objects: objects, pipe: pipe}
}

// Negotiate records a pending upload, starts the workflow (which blocks waiting
// for the completion signal), and returns a presigned PUT URL for the client.
func (s *Service) Negotiate(ctx context.Context, filename, contentType, userID string) (uploadID, url string, err error) {
	newID, err := id.New()
	if err != nil {
		return "", "", err
	}
	if err := s.repo.Create(ctx, newID, filename, contentType, "pending", userID); err != nil {
		return "", "", fmt.Errorf("create metadata: %w", err)
	}
	if err := s.pipe.Start(ctx, newID, contentType, filename); err != nil {
		return "", "", fmt.Errorf("start workflow: %w", err)
	}
	url, err = s.objects.Presign(ctx, newID+"/original", "PUT", presignTTL)
	if err != nil {
		return "", "", fmt.Errorf("presign: %w", err)
	}
	return newID, url, nil
}

// Complete signals the workflow that the client has finished uploading the bytes.
func (s *Service) Complete(ctx context.Context, id string) error {
	return s.pipe.Signal(ctx, id)
}

// PresignTTLSeconds is the presigned URL lifetime in seconds, for the API response.
func PresignTTLSeconds() int { return int(presignTTL.Seconds()) }
