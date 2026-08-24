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

	s := grpc.NewServer(grpc.MaxRecvMsgSize(20 * 1024 * 1024))
	imageprocess.RegisterResizerServer(s, &imaging.Server{})

	log.Println("image service listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
