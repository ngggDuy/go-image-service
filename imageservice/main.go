package main

import (
	"bytes"
	"context"
	"go-image-service/gen/imageprocess"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"net"

	"golang.org/x/image/draw"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// resizerServer implements the generated imageprocess.ResizerServer interface
type resizerServer struct {
	imageprocess.UnimplementedResizerServer
}

// Resize RPC method that HTTP server will call: image bytes in -> image bytes out
func (s *resizerServer) Resize(ctx context.Context, req *imageprocess.ResizeRequest) (*imageprocess.ResizeResponse, error) {
	// 1. Decode incoming bytes
	src, format, err := image.Decode(bytes.NewReader(req.GetImageToResize()))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "decode: %v", err)
	}

	// 2. Create blank destination canvas from requested image size to scale src into
	w := int(req.GetImageWidth())
	h := int(req.GetImageHeight())
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	// 3. Encode resized image back into original format byte stream
	var buf bytes.Buffer
	switch format {
	case "png":
		err = png.Encode(&buf, dst)
	default: // "jpeg"
		err = jpeg.Encode(&buf, dst, nil)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode: %v", err)
	}

	// 4. return resized image in response message
	return &imageprocess.ResizeResponse{ResizedImage: buf.Bytes()}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051") // 1. open TCP socket on conventional gRPC port
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer(grpc.MaxRecvMsgSize(20 * 1024 * 1024)) // 2. Create gRPC server with extended upload cap
	imageprocess.RegisterResizerServer(s, &resizerServer{})    // 3. implement generated interface

	log.Println("image service listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
