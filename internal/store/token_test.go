package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCredentialFileName(t *testing.T) {
	tests := []struct {
		name  string
		email string
		plan  string
		hash  string
		want  string
	}{
		{
			name:  "no plan",
			email: " you@example.com ",
			want:  "codex-you@example.com.json",
		},
		{
			name:  "plus",
			email: "you@example.com",
			plan:  "plus",
			want:  "codex-you@example.com-plus.json",
		},
		{
			name:  "normalized plan",
			email: "you@example.com",
			plan:  "chatgpt pro",
			want:  "codex-you@example.com-chatgpt-pro.json",
		},
		{
			name:  "team",
			email: "you@example.com",
			plan:  "team",
			hash:  "abc12345",
			want:  "codex-abc12345-you@example.com-team.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CredentialFileName(tt.email, tt.plan, tt.hash); got != tt.want {
				t.Fatalf("CredentialFileName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "codex-you@example.com.json")
	token := &CodexToken{
		IDToken:      "id",
		AccessToken:  "access",
		RefreshToken: "refresh",
		AccountID:    "account",
		Email:        "you@example.com",
		Expire:       "2026-05-14T00:00:00Z",
	}

	if err := token.Save(path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0700 {
		t.Fatalf("dir mode = %o, want 0700", got)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0600 {
		t.Fatalf("file mode = %o, want 0600", got)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Type != "codex" {
		t.Fatalf("Type = %q, want codex", got.Type)
	}
	if got.Email != token.Email || got.AccessToken != token.AccessToken || got.RefreshToken != token.RefreshToken {
		t.Fatalf("loaded token mismatch: %+v", got)
	}
}

func TestSaveTightensExistingFileMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "codex-you@example.com.json")
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	if err := (&CodexToken{Email: "you@example.com"}).Save(path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("file mode = %o, want 0600", got)
	}
}
