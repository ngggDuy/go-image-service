package metadata

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when no upload row exists for an id.
var ErrNotFound = errors.New("upload not found")

// Upload is one row of the uploads table, as returned by List.
type Upload struct {
	ID          string
	Filename    string
	ContentType string
	Status      string
	CreatedAt   time.Time
}

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

// List returns every upload owned by userID, newest first. An owner with no
// uploads yields an empty slice, not an error.
func (r *Repository) List(ctx context.Context, userID string) ([]Upload, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, original_filename, content_type, status, created_at
		   FROM uploads
		  WHERE user_id = $1
		  ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var uploads []Upload
	for rows.Next() {
		var u Upload
		if err := rows.Scan(&u.ID, &u.Filename, &u.ContentType, &u.Status, &u.CreatedAt); err != nil {
			return nil, err
		}
		uploads = append(uploads, u)
	}
	return uploads, rows.Err()
}
