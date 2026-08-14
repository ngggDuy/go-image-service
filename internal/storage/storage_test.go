package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// --- New ---

func TestNew_CreatesRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "uploads") // does not exist yet
	if _, err := New(root); err != nil {
		t.Fatalf("New() error: %v", err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		t.Errorf("New() did not create root dir %q (err=%v)", root, err)
	}
}

func TestNew_ExistingRootIsOK(t *testing.T) {
	root := t.TempDir() // already exists
	if _, err := New(root); err != nil {
		t.Errorf("New() on existing dir returned error: %v", err)
	}
}

func TestNew_ErrorOnBadRoot(t *testing.T) {
	// Put a FILE where a parent dir would need to be, so MkdirAll fails.
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	badRoot := filepath.Join(file, "sub") // can't create a dir under a file
	if _, err := New(badRoot); err == nil {
		t.Errorf("New(%q) expected error, got nil", badRoot)
	}
}

// --- Save ---

func TestSave_WritesContents(t *testing.T) {
	s := mustNew(t)
	want := []byte("hello bytes")
	if err := s.Save("abc", "original", ".jpg", want); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	got := readFile(t, filepath.Join(s.root, "abc", "original.jpg"))
	if string(got) != string(want) {
		t.Errorf("contents = %q, want %q", got, want)
	}
}

func TestSave_CreatesPerIDDir(t *testing.T) {
	s := mustNew(t)
	if err := s.Save("newid", "original", ".png", []byte("x")); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	info, err := os.Stat(filepath.Join(s.root, "newid"))
	if err != nil || !info.IsDir() {
		t.Errorf("Save() did not create the id directory (err=%v)", err)
	}
}

func TestSave_MultipleSizesCoexist(t *testing.T) {
	s := mustNew(t)
	if err := s.Save("id1", "original", ".jpg", []byte("orig")); err != nil {
		t.Fatal(err)
	}
	if err := s.Save("id1", "12x12", ".jpg", []byte("small")); err != nil {
		t.Fatal(err)
	}
	if got := string(readFile(t, filepath.Join(s.root, "id1", "original.jpg"))); got != "orig" {
		t.Errorf("original.jpg = %q, want %q", got, "orig")
	}
	if got := string(readFile(t, filepath.Join(s.root, "id1", "12x12.jpg"))); got != "small" {
		t.Errorf("12x12.jpg = %q, want %q", got, "small")
	}
}

func TestSave_Overwrites(t *testing.T) {
	s := mustNew(t)
	if err := s.Save("id", "original", ".jpg", []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := s.Save("id", "original", ".jpg", []byte("second")); err != nil {
		t.Fatalf("Save() overwrite error: %v", err)
	}
	if got := string(readFile(t, filepath.Join(s.root, "id", "original.jpg"))); got != "second" {
		t.Errorf("after overwrite = %q, want %q", got, "second")
	}
}

func TestSave_ErrorWhenIDPathIsAFile(t *testing.T) {
	s := mustNew(t)
	// Create a file exactly where the id directory would need to go.
	if err := os.WriteFile(filepath.Join(s.root, "clash"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.Save("clash", "original", ".jpg", []byte("y")); err == nil {
		t.Errorf("Save() expected error when id path is a file, got nil")
	}
}

// --- Path ---

func TestPath_ReturnsSavedFile(t *testing.T) {
	s := mustNew(t)
	if err := s.Save("abc", "original", ".jpg", []byte("x")); err != nil {
		t.Fatal(err)
	}
	got, err := s.Path("abc", "original")
	if err != nil {
		t.Fatalf("Path() error: %v", err)
	}
	want := filepath.Join(s.root, "abc", "original.jpg")
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestPath_FindsRegardlessOfExtension(t *testing.T) {
	s := mustNew(t)
	if err := s.Save("abc", "original", ".png", []byte("x")); err != nil { // saved as .png
		t.Fatal(err)
	}
	got, err := s.Path("abc", "original")
	if err != nil {
		t.Fatalf("Path() error: %v", err)
	}
	if filepath.Ext(got) != ".png" {
		t.Errorf("Path() ext = %q, want .png", filepath.Ext(got))
	}
}

func TestPath_DistinguishesSizes(t *testing.T) {
	s := mustNew(t)
	if err := s.Save("id", "original", ".jpg", []byte("orig")); err != nil {
		t.Fatal(err)
	}
	if err := s.Save("id", "12x12", ".jpg", []byte("small")); err != nil {
		t.Fatal(err)
	}
	got, err := s.Path("id", "12x12")
	if err != nil {
		t.Fatalf("Path() error: %v", err)
	}
	if filepath.Base(got) != "12x12.jpg" {
		t.Errorf("Path() base = %q, want 12x12.jpg", filepath.Base(got))
	}
}

func TestPath_NotFound(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, s *Store)
		id, size string
	}{
		{
			name:  "unknown id",
			setup: func(t *testing.T, s *Store) {},
			id:    "missing", size: "original",
		},
		{
			name: "known id but missing size",
			setup: func(t *testing.T, s *Store) {
				if err := s.Save("id", "original", ".jpg", []byte("x")); err != nil {
					t.Fatal(err)
				}
			},
			id: "id", size: "12x12",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := mustNew(t)
			tc.setup(t, s)
			_, err := s.Path(tc.id, tc.size)
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("Path() error = %v, want ErrNotFound", err)
			}
		})
	}
}

// --- helpers ---

func mustNew(t *testing.T) *Store {
	t.Helper()
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return s
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return b
}
