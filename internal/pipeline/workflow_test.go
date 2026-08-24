package pipeline

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/testsuite"
)

//Tests run real UploadWorkflow in Temporal's in-memory test
// environment, with the activities MOCKED, i.e.
// we verify the orchestration (fan-out + completion) without touching disk, gRPC, or Postgres.

func TestUploadWorkflow_Success(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	var a *Activities
	env.OnActivity(a.Resize, mock.Anything, mock.Anything).Return(nil).Times(len(sizes))
	env.OnActivity(a.Complete, mock.Anything, mock.Anything).Return(nil).Once()

	env.ExecuteWorkflow(UploadWorkflow, UploadInput{ID: "abc123", Ext: ".jpg", Filename: "x.jpg"})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow returned error: %v", err)
	}
	env.AssertExpectations(t)
}

func TestUploadWorkflow_ResizeFailsMarksFailed(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	var a *Activities
	env.OnActivity(a.Resize, mock.Anything, mock.Anything).Return(errors.New("resize boom"))
	env.OnActivity(a.Fail, mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(UploadWorkflow, UploadInput{ID: "abc123", Ext: ".jpg", Filename: "x.jpg"})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if env.GetWorkflowError() == nil {
		t.Fatal("expected the workflow to return an error, got nil")
	}
}
