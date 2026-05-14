package probe

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestStatusString(t *testing.T) {
	tests := map[Status]string{
		StatusValid:        "valid",
		StatusQuota:        "quota_exceeded",
		StatusUnauthorized: "unauthorized",
		StatusConnErr:      "conn_error",
		Status(99):         "conn_error",
	}
	for status, want := range tests {
		if got := status.String(); got != want {
			t.Fatalf("%v.String() = %q, want %q", status, got, want)
		}
	}
}

func TestCheckTokenStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   Status
	}{
		{name: "ok", status: http.StatusOK, want: StatusValid},
		{name: "quota", status: http.StatusTooManyRequests, want: StatusQuota},
		{name: "unauthorized", status: http.StatusUnauthorized, want: StatusUnauthorized},
		{name: "bad request invalid value", status: http.StatusBadRequest, body: `{"error":"invalid_value"}`, want: StatusValid},
		{name: "bad request other", status: http.StatusBadRequest, body: `{"error":"bad"}`, want: StatusConnErr},
		{name: "usage limit body", status: http.StatusInternalServerError, body: "usage limit reached", want: StatusQuota},
		{name: "quota body", status: http.StatusServiceUnavailable, body: "quota exhausted", want: StatusQuota},
		{name: "unknown", status: http.StatusServiceUnavailable, body: "try later", want: StatusConnErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotAuth string
			client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				gotAuth = r.Header.Get("Authorization")
				if r.Method != http.MethodPost {
					t.Fatalf("method = %s, want POST", r.Method)
				}
				return &http.Response{
					StatusCode: tt.status,
					Body:       io.NopCloser(strings.NewReader(tt.body)),
					Header:     make(http.Header),
				}, nil
			})}

			got := checkToken("token123", "https://example.test/codex", client)
			if got != tt.want {
				t.Fatalf("checkToken() = %v, want %v", got, tt.want)
			}
			if gotAuth != "Bearer token123" {
				t.Fatalf("Authorization = %q, want Bearer token123", gotAuth)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestCheckTokenConnectionError(t *testing.T) {
	client := &http.Client{Timeout: time.Second}
	if got := checkToken("token", "://bad-url", client); got != StatusConnErr {
		t.Fatalf("checkToken() = %v, want %v", got, StatusConnErr)
	}
}

func TestCheckTokenRequestBodyIncludesInstructions(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var payload map[string]any
		if err = json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if _, ok := payload["instructions"]; !ok {
			t.Fatal("request body missing 'instructions' field")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})}

	if got := checkToken("token", "https://example.test/codex", client); got != StatusValid {
		t.Fatalf("checkToken() = %v, want valid", got)
	}
}

func TestContainsAny(t *testing.T) {
	if !containsAny("quota exhausted", "usage limit", "quota") {
		t.Fatal("containsAny() = false, want true")
	}
	if containsAny("all good", "usage limit", "quota") {
		t.Fatal("containsAny() = true, want false")
	}
}
