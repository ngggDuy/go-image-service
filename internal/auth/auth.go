package auth

import (
	"context"

	"go-image-service/internal/id"
	"go-image-service/internal/user"
)

// UserRepo interface for auth service, satisifed by user.Repository
type UserRepo interface {
	Create(ctx context.Context, id, email, passwordHash string) error
	GetUserByEmail(ctx context.Context, email string) (user.User, error)
}

type Service struct {
	users UserRepo
}

func NewService(users UserRepo) *Service {
	return &Service{users: users}
}

// Register hashes the password, creates a user, and returns the new user id.
func (s *Service) Register(ctx context.Context, email, password string) (string, error) {
	// 1. hash the password
	hash, err := HashPassword(password)
	if err != nil {
		return "", err
	}

	// 2. generate new user id
	newUserID, err := id.New()
	if err != nil {
		return "", err
	}

	// 3. store new user
	if err := s.users.Create(ctx, newUserID, email, hash); err != nil {
		return "", err
	}

	// 4. return ID
	return newUserID, nil
}
