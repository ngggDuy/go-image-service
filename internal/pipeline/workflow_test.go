package pipeline

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/testsuite"
)

// These run the real UploadWorkflow in Temporal's in-memory test environment,
// with the activities mocked — verifying the signal/timer orchestration without
// real object storage, gRPC, or Postgres.

func TestUploadWorkflow_SignalThenComplete(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	var a *Activities
	env.OnActivity(a.Validate, mock.Anything, mock.Anything).Return(nil).Once()
	env.OnActivity(a.Processing, mock.Anything, mock.Anything).Return(nil).Once()
	env.OnActivity(a.Resize, mock.Anything, mock.Anything).Return(nil).Times(len(sizes))
	env.OnActivity(a.Complete, mock.Anything, mock.Anything).Return(nil).Once()

	// Deliver the completion signal shortly after the workflow starts.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(completeSignal, nil)
	}, time.Second)

	env.ExecuteWorkflow(UploadWorkflow, UploadInput{ID: "abc123", ContentType: "image/jpeg", Filename: "x.jpg"})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}
	env.AssertExpectations(t)
}

func TestUploadWorkflow_TimeoutAbandons(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	var a *Activities
	env.OnActivity(a.Abandon, mock.Anything, mock.Anything).Return(nil).Once()

	// No signal is ever sent → the timer fires (the test env auto-advances time)
	// → the workflow abandons the upload.
	env.ExecuteWorkflow(UploadWorkflow, UploadInput{ID: "abc123", ContentType: "image/jpeg"})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}
	env.AssertExpectations(t)
}
