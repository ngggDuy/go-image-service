package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound sentinal returns when no user found.
var ErrNotFound = errors.New("user not found")

// Repository stores user data for authN & authZ
type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// User is a registered user. Fields are exported so the auth package can read them.
type User struct {
	ID           string
	Email        string
	PasswordHash string
}

// Create stores a new user on initial sign up
func (r *Repository) Create(ctx context.Context, id, email, password_hash string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)`,
		id, email, password_hash,
	)
	return err
}

// GetUserByEmail returns user by given email, or ErrNotFound
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := r.db.QueryRow(ctx, `SELECT id, email, password_hash FROM users WHERE email = $1`, email).Scan(&u.ID, &u.Email, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return u, nil
}
