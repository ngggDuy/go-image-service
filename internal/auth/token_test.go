package auth

import "testing"

func TestToken_RoundTrip(t *testing.T) {
	secret := []byte("test-secret")

	token, err := IssueToken("user123", secret)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	got, err := ParseToken(token, secret)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if got != "user123" {
		t.Errorf("subject = %q, want %q", got, "user123")
	}
}

func TestToken_WrongSecretRejected(t *testing.T) {
	token, _ := IssueToken("user123", []byte("secret-a"))
	if _, err := ParseToken(token, []byte("secret-b")); err == nil {
		t.Error("expected error verifying with the wrong secret, got nil")
	}
}

func TestToken_TamperedRejected(t *testing.T) {
	token, _ := IssueToken("user123", []byte("secret"))
	if _, err := ParseToken(token+"x", []byte("secret")); err == nil {
		t.Error("expected error for a tampered token, got nil")
	}
}
