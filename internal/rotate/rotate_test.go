package rotate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/router-for-me/codex-rotator/internal/probe"
	"github.com/router-for-me/codex-rotator/internal/store"
)

func TestReadCurrentToken(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(validPath, []byte(`{"tokens":{"access_token":"access123"}}`), 0600); err != nil {
		t.Fatalf("write valid auth: %v", err)
	}
	if got := ReadCurrentToken(validPath); got != "access123" {
		t.Fatalf("ReadCurrentToken() = %q, want access123", got)
	}

	invalidPath := filepath.Join(dir, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("{"), 0600); err != nil {
		t.Fatalf("write invalid auth: %v", err)
	}
	if got := ReadCurrentToken(invalidPath); got != "" {
		t.Fatalf("ReadCurrentToken(invalid) = %q, want empty", got)
	}
	if got := ReadCurrentToken(filepath.Join(dir, "missing.json")); got != "" {
		t.Fatalf("ReadCurrentToken(missing) = %q, want empty", got)
	}
}

func TestWriteCodexAuth(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	token := &store.CodexToken{
		IDToken:      "id",
		AccessToken:  "access",
		RefreshToken: "refresh",
		AccountID:    "account",
	}

	if err := WriteCodexAuth(path, token); err != nil {
		t.Fatalf("WriteCodexAuth() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat auth: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("mode = %o, want 0600", got)
	}

	var auth codexAuthFile
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read auth: %v", err)
	}
	if err = json.Unmarshal(data, &auth); err != nil {
		t.Fatalf("unmarshal auth: %v", err)
	}
	if auth.AuthMode != "chatgpt" || auth.OpenAIAPIKey != nil {
		t.Fatalf("unexpected auth metadata: %+v", auth)
	}
	if auth.Tokens.AccessToken != "access" || auth.Tokens.IDToken != "id" || auth.Tokens.RefreshToken != "refresh" || auth.Tokens.AccountID != "account" {
		t.Fatalf("unexpected token block: %+v", auth.Tokens)
	}
	if _, err = time.Parse("2006-01-02T15:04:05.000Z", auth.LastRefresh); err != nil {
		t.Fatalf("LastRefresh is not parseable: %v", err)
	}
}

func TestWriteCodexAuthTightensExistingFileMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := WriteCodexAuth(path, &store.CodexToken{}); err != nil {
		t.Fatalf("WriteCodexAuth() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat auth: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("mode = %o, want 0600", got)
	}
}

func TestOnceCurrentTokenValidNoop(t *testing.T) {
	dir := t.TempDir()
	authPath := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"tokens":{"access_token":"current"}}`), 0600); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	poolDir := filepath.Join(dir, "pool")
	if err := (&store.CodexToken{Email: "pool@example.com", AccessToken: "pool"}).Save(filepath.Join(poolDir, "codex-pool@example.com.json")); err != nil {
		t.Fatalf("save token: %v", err)
	}

	rotated, email, err := onceWithChecker(authPath, poolDir, func(token string) probe.Status {
		if token != "current" {
			t.Fatalf("checked unexpected token %q", token)
		}
		return probe.StatusValid
	})
	if err != nil {
		t.Fatalf("onceWithChecker() error = %v", err)
	}
	if rotated || email != "" {
		t.Fatalf("onceWithChecker() = (%v, %q), want no-op", rotated, email)
	}
}

func TestOnceRotatesToFirstWorkingToken(t *testing.T) {
	dir := t.TempDir()
	authPath := filepath.Join(dir, "auth.json")
	poolDir := filepath.Join(dir, "pool")
	tokens := []*store.CodexToken{
		{Email: "empty@example.com"},
		{Email: "quota@example.com", AccessToken: "quota"},
		{Email: "valid@example.com", AccessToken: "valid", IDToken: "id", RefreshToken: "refresh", AccountID: "account"},
	}
	for _, token := range tokens {
		if err := token.Save(filepath.Join(poolDir, store.CredentialFileName(token.Email, "", ""))); err != nil {
			t.Fatalf("save %s: %v", token.Email, err)
		}
	}

	rotated, email, err := onceWithChecker(authPath, poolDir, func(token string) probe.Status {
		if token == "valid" {
			return probe.StatusValid
		}
		return probe.StatusQuota
	})
	if err != nil {
		t.Fatalf("onceWithChecker() error = %v", err)
	}
	if !rotated || email != "valid@example.com" {
		t.Fatalf("onceWithChecker() = (%v, %q), want rotate to valid@example.com", rotated, email)
	}
	if got := ReadCurrentToken(authPath); got != "valid" {
		t.Fatalf("active token = %q, want valid", got)
	}
}

func TestOnceErrors(t *testing.T) {
	t.Run("empty pool", func(t *testing.T) {
		dir := t.TempDir()
		_, _, err := onceWithChecker(filepath.Join(dir, "auth.json"), filepath.Join(dir, "missing-pool"), func(string) probe.Status {
			return probe.StatusConnErr
		})
		if err == nil || !strings.Contains(err.Error(), "no accounts in pool") {
			t.Fatalf("error = %v, want no accounts in pool", err)
		}
	})

	t.Run("no working account", func(t *testing.T) {
		dir := t.TempDir()
		poolDir := filepath.Join(dir, "pool")
		if err := (&store.CodexToken{Email: "quota@example.com", AccessToken: "quota"}).Save(filepath.Join(poolDir, "codex-quota@example.com.json")); err != nil {
			t.Fatalf("save token: %v", err)
		}
		_, _, err := onceWithChecker(filepath.Join(dir, "auth.json"), poolDir, func(string) probe.Status {
			return probe.StatusQuota
		})
		if err == nil || !strings.Contains(err.Error(), "no working account found") {
			t.Fatalf("error = %v, want no working account found", err)
		}
	})

	t.Run("write failure", func(t *testing.T) {
		dir := t.TempDir()
		authPath := filepath.Join(dir, "auth-dir")
		if err := os.Mkdir(authPath, 0700); err != nil {
			t.Fatalf("mkdir auth path: %v", err)
		}
		poolDir := filepath.Join(dir, "pool")
		if err := (&store.CodexToken{Email: "valid@example.com", AccessToken: "valid"}).Save(filepath.Join(poolDir, "codex-valid@example.com.json")); err != nil {
			t.Fatalf("save token: %v", err)
		}
		rotated, email, err := onceWithChecker(authPath, poolDir, func(string) probe.Status {
			return probe.StatusValid
		})
		if err == nil || !strings.Contains(err.Error(), "writing codex auth") {
			t.Fatalf("error = %v, want writing codex auth", err)
		}
		if rotated || email != "valid@example.com" {
			t.Fatalf("result = (%v, %q), want failed write with email", rotated, email)
		}
	})
}
