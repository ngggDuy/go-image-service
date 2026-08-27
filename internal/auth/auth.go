package auth

import (
	"context"
	"errors"

	"go-image-service/internal/id"
	"go-image-service/internal/user"
)

// ErrInvalidCredentials is returned by Login for a bad email OR a bad password.
var ErrInvalidCredentials = errors.New("invalid email or password")

// UserRepo interface for auth service, satisfied by user.Repository
type UserRepo interface {
	Create(ctx context.Context, id, email, passwordHash string) error
	GetUserByEmail(ctx context.Context, email string) (user.User, error)
}

type Service struct {
	users     UserRepo
	jwtSecret []byte
}

func NewService(users UserRepo, jwtSecret []byte) *Service {
	return &Service{users: users, jwtSecret: jwtSecret}
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

// Login verifies the credentials and returns signed JWT on success.
func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	u, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return "", ErrInvalidCredentials // unknown email
		}
		return "", err
	}
	if err := CheckPassword(u.PasswordHash, password); err != nil {
		return "", ErrInvalidCredentials // wrong password
	}
	return IssueToken(u.ID, s.jwtSecret)
}

// VerifyToken checks a token and returns the user id it belongs to.
func (s *Service) VerifyToken(token string) (string, error) {
	return ParseToken(token, s.jwtSecret)
}
