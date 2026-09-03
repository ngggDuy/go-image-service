package pipeline

import (
	"context"
	"errors"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

// Orchestrator starts UploadWorkflow executions and signals them.
type Orchestrator struct {
	client    client.Client
	taskQueue string
}

func NewOrchestrator(c client.Client, taskQueue string) *Orchestrator {
	return &Orchestrator{client: c, taskQueue: taskQueue}
}

func workflowID(id string) string { return "upload-" + id }

// Start schedules the workflow, which then blocks waiting for the completion signal.
func (o *Orchestrator) Start(ctx context.Context, id, contentType, filename string) error {
	_, err := o.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        workflowID(id),
		TaskQueue: o.taskQueue,
	}, UploadWorkflow, UploadInput{ID: id, ContentType: contentType, Filename: filename})
	return err
}

// Signal tells the workflow the bytes were uploaded. If the workflow already
// finished (NotFound), the work is done, so we treat it as success.
func (o *Orchestrator) Signal(ctx context.Context, id string) error {
	err := o.client.SignalWorkflow(ctx, workflowID(id), "", completeSignal, nil)
	var notFound *serviceerror.NotFound
	if errors.As(err, &notFound) {
		return nil
	}
	return err
}
