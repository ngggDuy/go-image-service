package auth

import (
	"context"
	"errors"
	"testing"

	"go-image-service/internal/user"
)

// fakeUserRepo is an in-memory UserRepo — lets us test the whole auth flow
// with no database (the payoff of depending on the UserRepo interface).
type fakeUserRepo struct {
	byEmail map[string]user.User
}

func newFakeRepo() *fakeUserRepo {
	return &fakeUserRepo{byEmail: map[string]user.User{}}
}

func (f *fakeUserRepo) Create(ctx context.Context, id, email, passwordHash string) error {
	f.byEmail[email] = user.User{ID: id, Email: email, PasswordHash: passwordHash}
	return nil
}

func (f *fakeUserRepo) GetUserByEmail(ctx context.Context, email string) (user.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return user.User{}, user.ErrNotFound
	}
	return u, nil
}

func TestRegisterThenLogin(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, []byte("secret"))
	ctx := context.Background()

	id, err := svc.Register(ctx, "alice@example.com", "password123")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if len(id) != 32 {
		t.Errorf("id length = %d, want 32", len(id))
	}
	// The stored value must be a hash, never the plaintext password.
	if repo.byEmail["alice@example.com"].PasswordHash == "password123" {
		t.Fatal("password was stored in plaintext!")
	}

	token, err := svc.Login(ctx, "alice@example.com", "password123")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	// The issued token must map back to the same user.
	uid, err := svc.VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if uid != id {
		t.Errorf("token subject = %q, want %q", uid, id)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, []byte("secret"))
	svc.Register(context.Background(), "alice@example.com", "password123")

	_, err := svc.Login(context.Background(), "alice@example.com", "wrongpassword")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	svc := NewService(newFakeRepo(), []byte("secret"))

	_, err := svc.Login(context.Background(), "nobody@example.com", "whatever")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("err = %v, want ErrInvalidCredentials", err)
	}
}
