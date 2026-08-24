package resizer

import (
	"context"

	"go-image-service/gen/imageprocess"
)

// Client adapts the generated gRPC client to a plain bytes-in/bytes-out API,
// so the rest of the app never has to touch protobuf types.
type Client struct {
	grpc imageprocess.ResizerClient // This is an interface, so the dependency is swappable.
}

// NOTE: An interface can have many implementations. Refactor this so that it takes in a struct and
// then we can unit test on a mock struct instead of a fakegRPC.
// In clean code, a struct should NOT depend on another struct.
func New(grpc imageprocess.ResizerClient) *Client {
	return &Client{grpc: grpc}
}

// Resize sends the image to the image service and returns the resized bytes.
func (c *Client) Resize(ctx context.Context, data []byte, width, height int) ([]byte, error) {
	resp, err := c.grpc.Resize(ctx, &imageprocess.ResizeRequest{
		ImageToResize: data,
		ImageWidth:    int32(width),
		ImageHeight:   int32(height),
	})
	if err != nil {
		return nil, err
	}
	return resp.GetResizedImage(), nil
}
