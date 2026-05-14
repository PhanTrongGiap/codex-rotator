package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	openaiAuthURL  = "https://auth.openai.com/oauth/authorize"
	openaiTokenURL = "https://auth.openai.com/oauth/token"
	openaiClientID = "app_EMoamEEZ73f0CkXaXp7hrann"
	redirectURI    = "http://localhost:1455/auth/callback"
)

// GenerateAuthURL builds the OAuth authorization URL.
func GenerateAuthURL(state string, pkce *PKCECodes) string {
	params := url.Values{
		"client_id":                  {openaiClientID},
		"response_type":              {"code"},
		"redirect_uri":               {redirectURI},
		"scope":                      {"openid email profile offline_access"},
		"state":                      {state},
		"code_challenge":             {pkce.CodeChallenge},
		"code_challenge_method":      {"S256"},
		"prompt":                     {"login"},
		"id_token_add_organizations": {"true"},
		"codex_cli_simplified_flow":  {"true"},
	}
	return openaiAuthURL + "?" + params.Encode()
}

// GenerateState creates a random CSRF state string.
func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ExchangeCodeForTokens exchanges the authorization code for tokens.
func ExchangeCodeForTokens(ctx context.Context, code string, pkce *PKCECodes) (*TokenData, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {openaiClientID},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {pkce.CodeVerifier},
	}
	return doTokenRequest(ctx, data)
}

// RefreshTokens uses a refresh_token to get a new access token.
func RefreshTokens(ctx context.Context, refreshToken string) (*TokenData, error) {
	data := url.Values{
		"client_id":     {openaiClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"scope":         {"openid profile email"},
	}
	return doTokenRequest(ctx, data)
}

func doTokenRequest(ctx context.Context, data url.Values) (*TokenData, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", openaiTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request status %d: %s", resp.StatusCode, body)
	}

	var tr struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err = json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	td := &TokenData{
		IDToken:      tr.IDToken,
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		Expire:       time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second).Format(time.RFC3339),
	}

	if claims, err := parseJWT(tr.IDToken); err == nil {
		td.Email = claims.Email
		td.AccountID = claims.CodexAuth.ChatgptAccountID
	}

	return td, nil
}

// PlanTypeAndHash extracts plan type and short account hash from an id_token.
func PlanTypeAndHash(idToken string) (planType, hash string) {
	claims, err := parseJWT(idToken)
	if err != nil {
		return "", ""
	}
	planType = strings.TrimSpace(claims.CodexAuth.ChatgptPlanType)
	accountID := strings.TrimSpace(claims.CodexAuth.ChatgptAccountID)
	if accountID != "" {
		d := sha256.Sum256([]byte(accountID))
		hash = hex.EncodeToString(d[:])[:8]
	}
	return planType, hash
}
