package storage

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrNotFound is returned by Path when no file exists for id/size.
// HTTP layer maps this to a 404 via errors.Is
var ErrNotFound = errors.New("image not found")

// Persists images under a root dir on disk
type Store struct {
	root string
}

// MkdirAll(root) at startup
func New(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &Store{root: root}, nil
}

// writes data to <root>/<id>/<name><ext>, e.g. uploads/abc/original.jpg.
func (s *Store) Save(id, name, ext string, data []byte) error {
	dir := filepath.Join(s.root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name+ext), data, 0o644)
}

// Path returns the path to the stored file for id + size (e.g. "12x12"),
// or ErrNotFound if none exists.
func (s *Store) Path(id, size string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(s.root, id, size+".*"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", ErrNotFound
	}
	return matches[0], nil
}
