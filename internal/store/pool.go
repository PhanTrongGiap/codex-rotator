package store

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultPoolDir returns ~/.codex-rotator/auth
func DefaultPoolDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex-rotator", "auth")
}

// ListPool returns all CodexToken entries found in poolDir.
func ListPool(poolDir string) ([]*CodexToken, error) {
	entries, err := os.ReadDir(poolDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var tokens []*CodexToken
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "codex-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		t, err := Load(filepath.Join(poolDir, name))
		if err != nil {
			continue
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}
