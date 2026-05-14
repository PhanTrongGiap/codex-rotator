package oauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func TestGeneratePKCECodes(t *testing.T) {
	codes, err := GeneratePKCECodes()
	if err != nil {
		t.Fatalf("GeneratePKCECodes() error = %v", err)
	}
	if len(codes.CodeVerifier) != 128 {
		t.Fatalf("verifier length = %d, want 128", len(codes.CodeVerifier))
	}
	if strings.Contains(codes.CodeVerifier, "=") || strings.Contains(codes.CodeChallenge, "=") {
		t.Fatalf("PKCE values must not contain padding: %+v", codes)
	}
	if matched := regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(codes.CodeVerifier); !matched {
		t.Fatalf("verifier is not URL-safe: %q", codes.CodeVerifier)
	}
	if got, want := codes.CodeChallenge, generateCodeChallenge(codes.CodeVerifier); got != want {
		t.Fatalf("challenge = %q, want %q", got, want)
	}
}

func TestGenerateState(t *testing.T) {
	state, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error = %v", err)
	}
	if matched := regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(state); !matched {
		t.Fatalf("state = %q, want 32 lowercase hex chars", state)
	}
}

func TestGenerateAuthURL(t *testing.T) {
	authURL := GenerateAuthURL("state123", &PKCECodes{CodeChallenge: "challenge123"})
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	q := parsed.Query()

	checks := map[string]string{
		"client_id":             openaiClientID,
		"response_type":         "code",
		"redirect_uri":          redirectURI,
		"state":                 "state123",
		"code_challenge":        "challenge123",
		"code_challenge_method": "S256",
		"prompt":                "login",
	}
	for key, want := range checks {
		if got := q.Get(key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
	if got := q.Get("scope"); !strings.Contains(got, "offline_access") || !strings.Contains(got, "email") {
		t.Fatalf("scope = %q, want email and offline_access", got)
	}
}

func TestParseJWTAndPlanTypeAndHash(t *testing.T) {
	token := fakeJWT(t, map[string]any{
		"email": "you@example.com",
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": "account-123",
			"chatgpt_plan_type":  "Team",
		},
	})

	claims, err := parseJWT(token)
	if err != nil {
		t.Fatalf("parseJWT() error = %v", err)
	}
	if claims.Email != "you@example.com" {
		t.Fatalf("Email = %q, want you@example.com", claims.Email)
	}
	if claims.CodexAuth.ChatgptAccountID != "account-123" {
		t.Fatalf("AccountID = %q, want account-123", claims.CodexAuth.ChatgptAccountID)
	}

	plan, hash := PlanTypeAndHash(token)
	if plan != "Team" {
		t.Fatalf("plan = %q, want Team", plan)
	}
	if len(hash) != 8 {
		t.Fatalf("hash length = %d, want 8", len(hash))
	}
}

func TestParseJWTRejectsMalformedToken(t *testing.T) {
	if _, err := parseJWT("not-a-jwt"); err == nil {
		t.Fatal("parseJWT() error = nil, want error")
	}
	if plan, hash := PlanTypeAndHash("not-a-jwt"); plan != "" || hash != "" {
		t.Fatalf("PlanTypeAndHash() = (%q, %q), want empty values", plan, hash)
	}
}

func TestExchangeCodeForTokens(t *testing.T) {
	idToken := fakeJWT(t, map[string]any{
		"email": "you@example.com",
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": "account-123",
		},
	})

	restore := replaceDefaultClient(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Fatalf("Content-Type = %q, want application/x-www-form-urlencoded", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		checks := map[string]string{
			"grant_type":    "authorization_code",
			"client_id":     openaiClientID,
			"code":          "code123",
			"redirect_uri":  redirectURI,
			"code_verifier": "verifier123",
		}
		for key, want := range checks {
			if got := r.Form.Get(key); got != want {
				t.Fatalf("%s = %q, want %q", key, got, want)
			}
		}
		return jsonResponse(http.StatusOK, map[string]any{
			"access_token":  "access",
			"refresh_token": "refresh",
			"id_token":      idToken,
			"expires_in":    3600,
		})
	})
	defer restore()

	td, err := ExchangeCodeForTokens(context.Background(), "code123", &PKCECodes{CodeVerifier: "verifier123"})
	if err != nil {
		t.Fatalf("ExchangeCodeForTokens() error = %v", err)
	}
	if td.AccessToken != "access" || td.RefreshToken != "refresh" || td.IDToken != idToken {
		t.Fatalf("unexpected token data: %+v", td)
	}
	if td.Email != "you@example.com" || td.AccountID != "account-123" {
		t.Fatalf("claims not copied: %+v", td)
	}
}

func TestRefreshTokens(t *testing.T) {
	restore := replaceDefaultClient(t, func(r *http.Request) (*http.Response, error) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		checks := map[string]string{
			"grant_type":    "refresh_token",
			"client_id":     openaiClientID,
			"refresh_token": "refresh123",
			"scope":         "openid profile email",
		}
		for key, want := range checks {
			if got := r.Form.Get(key); got != want {
				t.Fatalf("%s = %q, want %q", key, got, want)
			}
		}
		return jsonResponse(http.StatusOK, map[string]any{"expires_in": 1})
	})
	defer restore()

	if _, err := RefreshTokens(context.Background(), "refresh123"); err != nil {
		t.Fatalf("RefreshTokens() error = %v", err)
	}
}

func TestTokenRequestErrors(t *testing.T) {
	t.Run("non ok status", func(t *testing.T) {
		restore := replaceDefaultClient(t, func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader("bad request")),
				Header:     make(http.Header),
			}, nil
		})
		defer restore()

		_, err := RefreshTokens(context.Background(), "refresh")
		if err == nil || !strings.Contains(err.Error(), "token request status 400") {
			t.Fatalf("error = %v, want status error", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		restore := replaceDefaultClient(t, func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("{")),
				Header:     make(http.Header),
			}, nil
		})
		defer restore()

		_, err := RefreshTokens(context.Background(), "refresh")
		if err == nil || !strings.Contains(err.Error(), "failed to parse token response") {
			t.Fatalf("error = %v, want parse error", err)
		}
	})
}

func fakeJWT(t *testing.T, payload map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "none", "typ": "JWT"})
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	enc := base64.RawURLEncoding
	return enc.EncodeToString(header) + "." + enc.EncodeToString(body) + ".sig"
}

func replaceDefaultClient(t *testing.T, fn func(*http.Request) (*http.Response, error)) func() {
	t.Helper()
	orig := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: oauthRoundTripFunc(fn)}
	return func() {
		http.DefaultClient = orig
	}
}

type oauthRoundTripFunc func(*http.Request) (*http.Response, error)

func (f oauthRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(status int, body map[string]any) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(string(data))),
		Header:     make(http.Header),
	}, nil
}
