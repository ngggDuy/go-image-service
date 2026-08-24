package imaging

import (
	"context"

	"go-image-service/gen/imageprocess"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server adapts the pure Resize function to the generated gRPC ResizerServer.
type Server struct {
	imageprocess.UnimplementedResizerServer
}

func (s *Server) Resize(ctx context.Context, req *imageprocess.ResizeRequest) (*imageprocess.ResizeResponse, error) {
	resized, err := Resize(req.GetImageToResize(), int(req.GetImageWidth()), int(req.GetImageHeight()))
	if err != nil {
		// A failure here means the bytes weren't a valid image we can process.
		return nil, status.Errorf(codes.InvalidArgument, "resize: %v", err)
	}
	return &imageprocess.ResizeResponse{ResizedImage: resized}, nil
}
