package imaging

import (
	"context"

	"go-image-service/gen/imageprocess"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server adapts the pure Resize function to the generated gRPC ResizerServer.
//
// Decoding a full-resolution image allocates a raw pixel buffer (width*height*4
// bytes) — for a very large image that can be gigabytes. Because the upload
// workflow fans out its resizes as parallel activities, several Resize RPCs can
// arrive at once and, decoding simultaneously, exhaust memory. `sem` bounds how
// many decodes run concurrently so the service stays within its memory budget;
// callers simply wait their turn instead of OOM-killing the process.
type Server struct {
	imageprocess.UnimplementedResizerServer
	sem chan struct{}
}

// NewServer returns a Server that runs at most maxConcurrent resizes at a time.
// maxConcurrent <= 0 is treated as 1.
func NewServer(maxConcurrent int) *Server {
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	return &Server{sem: make(chan struct{}, maxConcurrent)}
}

func (s *Server) Resize(ctx context.Context, req *imageprocess.ResizeRequest) (*imageprocess.ResizeResponse, error) {
	// Wait for a decode slot (or give up if the caller's context is cancelled).
	// A nil semaphore (zero-value Server, e.g. in tests) means "no limit".
	if s.sem != nil {
		select {
		case s.sem <- struct{}{}:
			defer func() { <-s.sem }()
		case <-ctx.Done():
			return nil, status.FromContextError(ctx.Err()).Err()
		}
	}

	resized, err := Resize(req.GetImageToResize(), int(req.GetImageWidth()), int(req.GetImageHeight()))
	if err != nil {
		// A failure here means the bytes weren't a valid image we can process.
		return nil, status.Errorf(codes.InvalidArgument, "resize: %v", err)
	}
	return &imageprocess.ResizeResponse{ResizedImage: resized}, nil
}
