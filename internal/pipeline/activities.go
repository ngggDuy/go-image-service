package pipeline

import (
	"context"

	"go-image-service/internal/metadata"
	"go-image-service/internal/resizer"
	"go-image-service/internal/storage"
)

// ResizeInput tells a resize activity which stored image to process and to what size.
type ResizeInput struct {
	ID     string
	Ext    string
	Name   string // output name, e.g. "12x12"
	Width  int
	Height int
}

// Activities holds the real dependencies the worker uses. The worker registers
// one instance; Temporal invokes its methods as activities.
type Activities struct {
	Resizer *resizer.Client
	Store   *storage.Store
	Repo    *metadata.Repository
}

// Resize reads the original from storage, resizes it via the image service, and
// stores the result. It is idempotent: re-running overwrites the same file.
func (a *Activities) Resize(ctx context.Context, in ResizeInput) error {
	data, err := a.Store.Read(in.ID, "original", in.Ext)
	if err != nil {
		return err
	}
	resized, err := a.Resizer.Resize(ctx, data, in.Width, in.Height)
	if err != nil {
		return err
	}
	return a.Store.Save(in.ID, in.Name, in.Ext, resized)
}

// Complete marks the upload finished.
func (a *Activities) Complete(ctx context.Context, id string) error {
	return a.Repo.UpdateStatus(ctx, id, "complete")
}

// Fail marks the upload failed.
func (a *Activities) Fail(ctx context.Context, id string) error {
	return a.Repo.UpdateStatus(ctx, id, "failed")
}
