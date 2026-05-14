package rotate

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/router-for-me/codex-rotator/internal/probe"
	"github.com/router-for-me/codex-rotator/internal/store"
)

// codexAuthFile is the format codex CLI reads.
type codexAuthFile struct {
	AuthMode     string     `json:"auth_mode"`
	OpenAIAPIKey *string    `json:"OPENAI_API_KEY"`
	Tokens       tokenBlock `json:"tokens"`
	LastRefresh  string     `json:"last_refresh"`
}

type tokenBlock struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    string `json:"account_id"`
}

// ReadCurrentToken reads the access_token from the codex auth file.
func ReadCurrentToken(codexAuthPath string) string {
	data, err := os.ReadFile(codexAuthPath)
	if err != nil {
		return ""
	}
	var af codexAuthFile
	if err = json.Unmarshal(data, &af); err != nil {
		return ""
	}
	return af.Tokens.AccessToken
}

// WriteCodexAuth overwrites the codex auth file with the given pool token.
func WriteCodexAuth(codexAuthPath string, t *store.CodexToken) error {
	af := codexAuthFile{
		AuthMode:     "chatgpt",
		OpenAIAPIKey: nil,
		Tokens: tokenBlock{
			AccessToken:  t.AccessToken,
			IDToken:      t.IDToken,
			RefreshToken: t.RefreshToken,
			AccountID:    t.AccountID,
		},
		LastRefresh: time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	data, err := json.MarshalIndent(af, "", "  ")
	if err != nil {
		return err
	}
	tmp := codexAuthPath + ".tmp"
	if err = os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, codexAuthPath)
}

// Once performs a single rotate cycle:
//  1. Check current token in codexAuthPath.
//  2. If valid → no-op. Otherwise scan pool for a working account → swap.
//
// Returns (rotated bool, email of new account, error).
func Once(codexAuthPath, poolDir string) (bool, string, error) {
	return onceWithChecker(codexAuthPath, poolDir, probe.CheckToken)
}

func onceWithChecker(codexAuthPath, poolDir string, checkToken func(string) probe.Status) (bool, string, error) {
	current := ReadCurrentToken(codexAuthPath)
	if current != "" {
		if s := checkToken(current); s == probe.StatusValid {
			return false, "", nil
		}
	}

	tokens, err := store.ListPool(poolDir)
	if err != nil {
		return false, "", fmt.Errorf("listing pool: %w", err)
	}
	if len(tokens) == 0 {
		return false, "", fmt.Errorf("no accounts in pool %s", poolDir)
	}

	for _, t := range tokens {
		if t.AccessToken == "" {
			continue
		}
		if s := checkToken(t.AccessToken); s == probe.StatusValid {
			if err = WriteCodexAuth(codexAuthPath, t); err != nil {
				return false, t.Email, fmt.Errorf("writing codex auth: %w", err)
			}
			return true, t.Email, nil
		}
	}

	return false, "", fmt.Errorf("no working account found in pool")
}
