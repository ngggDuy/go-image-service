package id

import (
	"encoding/hex"
	"testing"
)

func TestNew(t *testing.T) {
	got, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	if len(got) != 32 {
		t.Errorf("New() length = %d, want 32", len(got))
	}
	if _, err := hex.DecodeString(got); err != nil {
		t.Errorf("New() returned invalid hex")
	}
}

func TestNewUnique(t *testing.T) {
	first, _ := New()
	second, _ := New()

	if first == second {
		t.Fatalf("New() returned same value, content not unique.")
	}
}
