package main

import (
	"log"
	"net"

	"go-image-service/gen/imageprocess"
	"go-image-service/internal/config"
	"go-image-service/internal/imaging"

	"google.golang.org/grpc"
)

func main() {
	addr := config.ListenAddr("50051")
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(grpc.MaxRecvMsgSize(300 * 1024 * 1024)) // allow heavy images (> maxImageBytes + framing)
	// Serialize decodes: one full-res decode of a very large image can use
	// gigabytes of RAM, so we don't let the parallel resize activities decode
	// several at once and OOM the container.
	imageprocess.RegisterResizerServer(s, imaging.NewServer(1))

	log.Printf("image service listening on %s", addr)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
