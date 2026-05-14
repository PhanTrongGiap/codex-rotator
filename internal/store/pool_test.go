package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListPool(t *testing.T) {
	dir := t.TempDir()

	if err := (&CodexToken{Email: "a@example.com", AccessToken: "a"}).Save(filepath.Join(dir, "codex-a@example.com.json")); err != nil {
		t.Fatalf("save valid token: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "codex-bad.json"), []byte("{"), 0600); err != nil {
		t.Fatalf("write invalid json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "note.json"), []byte(`{"email":"ignored@example.com"}`), 0600); err != nil {
		t.Fatalf("write ignored file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "codex-dir.json"), 0700); err != nil {
		t.Fatalf("mkdir ignored dir: %v", err)
	}

	tokens, err := ListPool(dir)
	if err != nil {
		t.Fatalf("ListPool() error = %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("len(tokens) = %d, want 1", len(tokens))
	}
	if tokens[0].Email != "a@example.com" {
		t.Fatalf("Email = %q, want a@example.com", tokens[0].Email)
	}
}

func TestListPoolMissingDir(t *testing.T) {
	tokens, err := ListPool(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("ListPool() error = %v", err)
	}
	if len(tokens) != 0 {
		t.Fatalf("len(tokens) = %d, want 0", len(tokens))
	}
}
