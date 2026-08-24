package imaging

import (
	"context"
	"testing"

	"go-image-service/gen/imageprocess"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServerResize_OK(t *testing.T) {
	s := &Server{}
	src := makeImage(t, "png", 40, 40)

	resp, err := s.Resize(context.Background(), &imageprocess.ResizeRequest{
		ImageToResize: src,
		ImageWidth:    12,
		ImageHeight:   12,
	})
	if err != nil {
		t.Fatalf("Resize: %v", err)
	}
	if len(resp.GetResizedImage()) == 0 {
		t.Error("expected resized bytes, got empty")
	}
}

func TestServerResize_InvalidArgument(t *testing.T) {
	s := &Server{}

	_, err := s.Resize(context.Background(), &imageprocess.ResizeRequest{
		ImageToResize: []byte("not an image"),
		ImageWidth:    12,
		ImageHeight:   12,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("error code = %v, want InvalidArgument", status.Code(err))
	}
}
