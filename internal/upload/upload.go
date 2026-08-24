package upload

import (
	"context"

	"go-image-service/internal/id"
)

// Store saves image bytes under an id and a name (e.g. "original").
type Store interface {
	Save(id, name, ext string, data []byte) error
}

// Repository persists metadata about an upload.
type Repository interface {
	Create(ctx context.Context, id, filename, ext, status string) error
}

// UploadStarter kicks off the asynchronous resize pipeline (a Temporal workflow
// in production). It returns as soon as the work is scheduled.
type UploadStarter interface {
	Start(ctx context.Context, id, ext, filename string) error
}

// Service handles an upload: it stores the original, records the upload as
// "processing", and starts the async resize pipeline. It does NOT wait for the
// resizes — those run in the workflow and flip the status to "complete".
type Service struct {
	store   Store
	repo    Repository
	starter UploadStarter
}

func New(s Store, repo Repository, starter UploadStarter) *Service {
	return &Service{store: s, repo: repo, starter: starter}
}

// Process stores the original, records "processing", and starts the pipeline.
// Returns the new upload id immediately (the caller responds 202 Accepted).
func (s *Service) Process(ctx context.Context, data []byte, filename, ext string) (string, error) {
	newID, err := id.New()
	if err != nil {
		return "", err
	}

	if err := s.store.Save(newID, "original", ext, data); err != nil {
		return "", err
	}

	if err := s.repo.Create(ctx, newID, filename, ext, "processing"); err != nil {
		return "", err
	}

	if err := s.starter.Start(ctx, newID, ext, filename); err != nil {
		return "", err
	}

	return newID, nil
}
