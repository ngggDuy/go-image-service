package resizer

import (
	"context"
	"errors"
	"testing"

	"go-image-service/gen/imageprocess"

	"google.golang.org/grpc"
)

// fakeGRPC implements imageprocess.ResizerClient for tests — no real server.
// It records the request it was called with and returns stub resp/err.
type fakeGRPC struct {
	gotReq *imageprocess.ResizeRequest
	resp   *imageprocess.ResizeResponse
	err    error
}

func (f *fakeGRPC) Resize(ctx context.Context, in *imageprocess.ResizeRequest, opts ...grpc.CallOption) (*imageprocess.ResizeResponse, error) {
	f.gotReq = in
	return f.resp, f.err
}

func TestResize_TranslatesAndReturnsBytes(t *testing.T) {
	fake := &fakeGRPC{resp: &imageprocess.ResizeResponse{ResizedImage: []byte("resized")}}
	c := New(fake) // The caller supplies the dependency. The type doesn't build it.

	got, err := c.Resize(context.Background(), []byte("orig"), 12, 25)
	if err != nil {
		t.Fatalf("Resize() error: %v", err)
	}
	if string(got) != "resized" {
		t.Errorf("Resize() = %q, want %q", got, "resized")
	}
	// This is a spy that records how it was called (gotReq), so we can assert that our code passed the right args.
	// It should have translated our args into the protobuf request correctly.
	if string(fake.gotReq.GetImageToResize()) != "orig" {
		t.Errorf("request image = %q, want %q", fake.gotReq.GetImageToResize(), "orig")
	}
	if fake.gotReq.GetImageWidth() != 12 || fake.gotReq.GetImageHeight() != 25 {
		t.Errorf("request dims = %dx%d, want 12x25", fake.gotReq.GetImageWidth(), fake.gotReq.GetImageHeight())
	}
}

func TestResize_PropagatesError(t *testing.T) {
	fake := &fakeGRPC{err: errors.New("boom")}
	c := New(fake)
	if _, err := c.Resize(context.Background(), []byte("x"), 1, 1); err == nil {
		t.Error("Resize() expected error, got nil")
	}
}
