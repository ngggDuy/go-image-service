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

// Create inserts a metadata record for one upload. Parameterized placeholders
// ($1..$4) keep user input from being interpreted as SQL.
func (r *Repository) Create(ctx context.Context, id, filename, ext, status string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO uploads (id, original_filename, ext, status)
		 VALUES ($1, $2, $3, $4)`,
		id, filename, ext, status,
	)
	return err
}

// UpdateStatus changes the status of an existing upload (e.g. processing -> complete).
func (r *Repository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.Exec(ctx, `UPDATE uploads SET status = $1 WHERE id = $2`, status, id)
	return err
}

// GetStatus returns the current status of an upload, or ErrNotFound.
func (r *Repository) GetStatus(ctx context.Context, id string) (string, error) {
	var status string
	err := r.db.QueryRow(ctx, `SELECT status FROM uploads WHERE id = $1`, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return status, err
}
