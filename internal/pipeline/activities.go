package pipeline

import (
	"context"
	"net/http"

	"go-image-service/internal/metadata"
	"go-image-service/internal/resizer"
	"go-image-service/internal/storage"

	"go.temporal.io/sdk/temporal"
)

const maxImageBytes = 200 * 1024 * 1024 // 200 MB — presigned uploads let us handle heavy images

type ValidateInput struct {
	ID          string
	ContentType string
}

type ResizeInput struct {
	ID          string
	ContentType string
	Name        string
	Width       int
	Height      int
}

// Activities holds the worker's real dependencies. Objects is the object store
// (an interface, so tests can fake it).
type Activities struct {
	Resizer *resizer.Client
	Objects storage.ObjectStore
	Repo    *metadata.Repository
}

func objectKey(id, name string) string { return id + "/" + name }

// Validate sniffs the uploaded object's type and checks its size. On a bad
// type/size it deletes the object, marks the upload "rejected", and returns a
// non-retryable error so Temporal doesn't retry a permanent rejection.
func (a *Activities) Validate(ctx context.Context, in ValidateInput) error {
	if err := a.Repo.UpdateStatus(ctx, in.ID, "validating"); err != nil {
		return err
	}
	orig := objectKey(in.ID, "original")

	sniff, err := a.Objects.GetFirstNBytes(ctx, orig, 512)
	if err != nil {
		return err
	}
	switch http.DetectContentType(sniff) {
	case "image/jpeg", "image/png":
		// ok
	default:
		return a.reject(ctx, in.ID, orig, "unsupported content type")
	}

	size, _, err := a.Objects.Head(ctx, orig)
	if err != nil {
		return err
	}
	if size > maxImageBytes {
		return a.reject(ctx, in.ID, orig, "image too large")
	}
	return nil
}

func (a *Activities) reject(ctx context.Context, id, key, reason string) error {
	_ = a.Objects.Delete(ctx, key)
	_ = a.Repo.UpdateStatus(ctx, id, "rejected")
	return temporal.NewNonRetryableApplicationError(reason, "rejected", nil)
}

// Resize reads the original from object storage, resizes it via the image
// service, and writes the result back. Idempotent (overwrites the same key).
func (a *Activities) Resize(ctx context.Context, in ResizeInput) error {
	data, err := a.Objects.Get(ctx, objectKey(in.ID, "original"))
	if err != nil {
		return err
	}
	resized, err := a.Resizer.Resize(ctx, data, in.Width, in.Height)
	if err != nil {
		return err
	}
	return a.Objects.Put(ctx, objectKey(in.ID, in.Name), resized, in.ContentType)
}

func (a *Activities) Processing(ctx context.Context, id string) error {
	return a.Repo.UpdateStatus(ctx, id, "processing")
}

func (a *Activities) Complete(ctx context.Context, id string) error {
	return a.Repo.UpdateStatus(ctx, id, "complete")
}

func (a *Activities) Fail(ctx context.Context, id string) error {
	return a.Repo.UpdateStatus(ctx, id, "failed")
}

func (a *Activities) Abandon(ctx context.Context, id string) error {
	_ = a.Objects.Delete(ctx, objectKey(id, "original"))
	return a.Repo.UpdateStatus(ctx, id, "abandoned")
}
