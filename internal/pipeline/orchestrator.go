package pipeline

import (
	"context"

	"go.temporal.io/sdk/client"
)

// Orchestrator starts UploadWorkflow executions
type Orchestrator struct {
	client    client.Client
	taskQueue string
}

func NewOrchestrator(c client.Client, taskQueue string) *Orchestrator {
	return &Orchestrator{client: c, taskQueue: taskQueue}
}

// Start schedules the workflow and returns immediately (async).
func (o *Orchestrator) Start(ctx context.Context, id, ext, filename string) error {
	_, err := o.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        "upload-" + id,
		TaskQueue: o.taskQueue,
	}, UploadWorkflow, UploadInput{ID: id, Ext: ext, Filename: filename})
	return err
}
