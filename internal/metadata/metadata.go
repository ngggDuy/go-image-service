package metadata

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when no upload row exists for an id.
var ErrNotFound = errors.New("upload not found")

// Repository stores and retrieves upload metadata in Postgres.
type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create inserts a metadata record for one upload, owned by userID.
func (r *Repository) Create(ctx context.Context, id, filename, contentType, status, userID string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO uploads (id, original_filename, content_type, status, user_id)
		 VALUES ($1, $2, $3, $4, $5)`,
		id, filename, contentType, status, userID,
	)
	return err
}

// UpdateStatus changes the status of an existing upload (e.g. pending -> complete).
func (r *Repository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.Exec(ctx, `UPDATE uploads SET status = $1 WHERE id = $2`, status, id)
	return err
}

// Get returns the status, owner (user id), and content type of an upload, or ErrNotFound.
func (r *Repository) Get(ctx context.Context, id string) (status, userID, contentType string, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT status, user_id, content_type FROM uploads WHERE id = $1`, id,
	).Scan(&status, &userID, &contentType)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", "", ErrNotFound
	}
	return status, userID, contentType, err
}
