package authgrpc

import (
	"context"
	"errors"

	"go-image-service/gen/authpb"
	"go-image-service/internal/auth"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server adapts the in-process auth.Service to the generated gRPC AuthServiceServer.
// All the real logic still lives in auth.Service; this just translates
// proto messages ↔ Go calls and Go errors ↔ gRPC status codes.
type Server struct {
	authpb.UnimplementedAuthServiceServer
	svc *auth.Service
}

func NewServer(svc *auth.Service) *Server {
	return &Server{svc: svc}
}

func (s *Server) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	id, err := s.svc.Register(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "register: %v", err)
	}
	return &authpb.RegisterResponse{UserId: id}, nil
}

func (s *Server) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	token, err := s.svc.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			// Unauthenticated is gRPC's 401 equivalent.
			return nil, status.Error(codes.Unauthenticated, "invalid email or password")
		}
		return nil, status.Errorf(codes.Internal, "login: %v", err)
	}
	return &authpb.LoginResponse{Token: token}, nil
}

func (s *Server) Verify(ctx context.Context, req *authpb.VerifyRequest) (*authpb.VerifyResponse, error) {
	userID, err := s.svc.VerifyToken(req.GetToken())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}
	return &authpb.VerifyResponse{UserId: userID}, nil
}
