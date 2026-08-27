package pipeline

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	completeSignal = "upload-complete"
	uploadTimeout  = 15 * time.Minute
)

// UploadInput carries only references (id + content type), never image bytes —
// the bytes live in object storage; activities read/write them there.
type UploadInput struct {
	ID          string
	ContentType string
	Filename    string
}

type resizeSpec struct {
	Name string
	W, H int
}

var sizes = []resizeSpec{
	{Name: "12x12", W: 12, H: 12},
	{Name: "25x25", W: 25, H: 25},
}

// UploadWorkflow starts BEFORE the bytes exist. It waits for the upload-complete
// signal (or abandons after a timeout), validates the uploaded object, resizes
// it in parallel, and marks the upload complete.
func UploadWorkflow(ctx workflow.Context, in UploadInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
	})
	var a *Activities

	// Wait for the client's completion signal, or give up after the timeout.
	timerCtx, cancelTimer := workflow.WithCancel(ctx)
	timer := workflow.NewTimer(timerCtx, uploadTimeout)
	sigCh := workflow.GetSignalChannel(ctx, completeSignal)

	signaled := false
	sel := workflow.NewSelector(ctx)
	sel.AddReceive(sigCh, func(c workflow.ReceiveChannel, _ bool) {
		c.Receive(ctx, nil)
		signaled = true
	})
	sel.AddFuture(timer, func(workflow.Future) {})
	sel.Select(ctx)

	if !signaled {
		return workflow.ExecuteActivity(ctx, a.Abandon, in.ID).Get(ctx, nil)
	}
	cancelTimer() // avoid a zombie 15-minute timer lingering in history

	// Validate — its own policy: a rejection is permanent, so don't retry it.
	valCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	})
	if err := workflow.ExecuteActivity(valCtx, a.Validate,
		ValidateInput{ID: in.ID, ContentType: in.ContentType}).Get(ctx, nil); err != nil {
		return err // Validate already set status "rejected" and deleted the object
	}

	if err := workflow.ExecuteActivity(ctx, a.Processing, in.ID).Get(ctx, nil); err != nil {
		return err
	}

	// Resize fan-out.
	futures := make([]workflow.Future, 0, len(sizes))
	for _, s := range sizes {
		futures = append(futures, workflow.ExecuteActivity(ctx, a.Resize, ResizeInput{
			ID: in.ID, ContentType: in.ContentType, Name: s.Name, Width: s.W, Height: s.H,
		}))
	}
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
