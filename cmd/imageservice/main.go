package main

import (
	"log"
	"net"

	"go-image-service/gen/imageprocess"
	"go-image-service/internal/imaging"

	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(grpc.MaxRecvMsgSize(300 * 1024 * 1024)) // allow heavy images (> maxImageBytes + framing)
	// Serialize decodes: one full-res decode of a very large image can use
	// gigabytes of RAM, so we don't let the parallel resize activities decode
	// several at once and OOM the container.
	imageprocess.RegisterResizerServer(s, imaging.NewServer(1))

	log.Println("image service listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
