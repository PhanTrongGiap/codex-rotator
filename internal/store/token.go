package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// CodexToken matches the file format written by CLIProxyAPI for codex accounts.
type CodexToken struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    string `json:"account_id"`
	LastRefresh  string `json:"last_refresh"`
	Email        string `json:"email"`
	Type         string `json:"type"`
	Expire       string `json:"expired"`
}

// Save writes the token to authFilePath, creating dirs as needed.
func (t *CodexToken) Save(authFilePath string) error {
	t.Type = "codex"
	if err := os.MkdirAll(filepath.Dir(authFilePath), 0700); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	f, err := os.OpenFile(authFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err = f.Chmod(0600); err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(t)
}

// Load reads a CodexToken from a JSON file.
func Load(path string) (*CodexToken, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t CodexToken
	if err = json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// CredentialFileName returns the standardised pool filename for an account.
func CredentialFileName(email, planType, hashAccountID string) string {
	email = strings.TrimSpace(email)
	plan := normalizePlan(planType)
	if plan == "" {
		return fmt.Sprintf("codex-%s.json", email)
	}
	if plan == "Team" {
		return fmt.Sprintf("codex-%s-%s-%s.json", hashAccountID, email, strings.ToLower(plan))
	}
	return fmt.Sprintf("codex-%s-%s.json", email, strings.ToLower(plan))
}

func normalizePlan(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	parts := strings.FieldsFunc(p, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for i, part := range parts {
		parts[i] = cases.Title(language.English).String(part)
	}
	return strings.Join(parts, "-")
}
