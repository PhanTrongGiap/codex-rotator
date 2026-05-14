package probe

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

const codexResponsesURL = "https://chatgpt.com/backend-api/codex/responses"

// Status represents the validity of a token.
type Status int

const (
	StatusValid        Status = iota
	StatusQuota               // 429 or usage-limit message
	StatusUnauthorized        // 401
	StatusConnErr             // network / unknown
)

func (s Status) String() string {
	switch s {
	case StatusValid:
		return "valid"
	case StatusQuota:
		return "quota_exceeded"
	case StatusUnauthorized:
		return "unauthorized"
	default:
		return "conn_error"
	}
}

// CheckToken probes whether the given access_token is usable.
func CheckToken(accessToken string) Status {
	return checkToken(accessToken, codexResponsesURL, &http.Client{Timeout: 12 * time.Second})
}

func checkToken(accessToken, endpoint string, client *http.Client) Status {
	payload := map[string]any{
		"model":        "gpt-5.3-codex",
		"stream":       true,
		"store":        false,
		"instructions": "",
		"input": []map[string]any{
			{
				"type": "message",
				"role": "user",
				"content": []map[string]string{
					{"type": "text", "text": "hi"},
				},
			},
		},
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return StatusConnErr
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Version", "1.0.0")
	req.Header.Set("Openai-Beta", "responses=experimental")
	req.Header.Set("User-Agent", "codex_cli_rs/1.0.0")

	resp, err := client.Do(req)
	if err != nil {
		return StatusConnErr
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return StatusValid
	case http.StatusTooManyRequests:
		return StatusQuota
	case http.StatusUnauthorized:
		return StatusUnauthorized
	case http.StatusBadRequest:
		// CLIProxyAPI pattern: 400 with "invalid_value" means token is fine, just payload wrong
		respBody, _ := io.ReadAll(resp.Body)
		if containsAny(string(respBody), "invalid_value", "invalid value") {
			return StatusValid
		}
		return StatusConnErr
	default:
		respBody, _ := io.ReadAll(resp.Body)
		if containsAny(string(respBody), "usage limit", "quota") {
			return StatusQuota
		}
		return StatusConnErr
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
