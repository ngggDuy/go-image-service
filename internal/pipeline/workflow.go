package pipeline

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// UploadInput is the workflow input. It carries only a REFERENCE to the already
// saved original (id + ext) — never the image bytes. Temporal stores inputs in
// workflow history and has payload size limits, so large blobs stay on disk and
// activities read/write them there.
type UploadInput struct {
	ID       string
	Ext      string
	Filename string
}

type resizeSpec struct {
	Name string
	W, H int
}

var sizes = []resizeSpec{
	{Name: "12x12", W: 12, H: 12},
	{Name: "25x25", W: 25, H: 25},
}

// UploadWorkflow resizes the uploaded image into every size IN PARALLEL, then
// marks the upload complete. If any resize ultimately fails, it marks it failed.
func UploadWorkflow(ctx workflow.Context, in UploadInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute, // generous cap for large images
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	})

	var a *Activities // nil receiver: used only to name the activities for lookup

	// Fan out: start every resize activity at once, collect their futures.
	futures := make([]workflow.Future, 0, len(sizes))
	for _, s := range sizes {
		f := workflow.ExecuteActivity(ctx, a.Resize, ResizeInput{
			ID: in.ID, Ext: in.Ext, Name: s.Name, Width: s.W, Height: s.H,
		})
		futures = append(futures, f)
	}

	// Fan in: wait for all of them.
	var resizeErr error
	for _, f := range futures {
		if err := f.Get(ctx, nil); err != nil {
			resizeErr = err
		}
	}

	if resizeErr != nil {
		_ = workflow.ExecuteActivity(ctx, a.Fail, in.ID).Get(ctx, nil)
		return resizeErr
	}

	return workflow.ExecuteActivity(ctx, a.Complete, in.ID).Get(ctx, nil)
}
