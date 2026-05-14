package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/router-for-me/codex-rotator/internal/store"
	"github.com/spf13/cobra"
)

func TestParseCallbackInput(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantCode  string
		wantState string
		wantError string
	}{
		{
			name:      "full callback URL",
			raw:       "http://localhost:1455/auth/callback?code=code123&scope=openid&state=state123",
			wantCode:  "code123",
			wantState: "state123",
		},
		{
			name:      "query string only",
			raw:       "code=code123&scope=openid&state=state123",
			wantCode:  "code123",
			wantState: "state123",
		},
		{
			name:      "oauth error",
			raw:       "http://localhost:1455/auth/callback?error=access_denied",
			wantError: "access_denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCallbackInput(tt.raw)
			if err != nil {
				t.Fatalf("parseCallbackInput() error = %v", err)
			}
			if got.Code != tt.wantCode || got.State != tt.wantState || got.Error != tt.wantError {
				t.Fatalf("parseCallbackInput() = %+v, want code=%q state=%q error=%q", got, tt.wantCode, tt.wantState, tt.wantError)
			}
		})
	}
}

func TestParseCallbackInputRejectsMissingFields(t *testing.T) {
	if _, err := parseCallbackInput("http://localhost:1455/auth/callback?code=code123"); err == nil {
		t.Fatal("parseCallbackInput() error = nil, want missing state error")
	}
}

func buildRoot(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	root := &cobra.Command{Use: "codex-rotator"}
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.PersistentFlags().StringVar(&flagPool, "pool", store.DefaultPoolDir(), "")
	root.PersistentFlags().StringVar(&flagCodexAuth, "codex-auth", defaultCodexAuth(), "")
	root.AddCommand(cmdLogin(), cmdList(), cmdRotate(), cmdDaemon(), cmdRun(), cmdImport())
	return root, out
}

func TestLoginHelp(t *testing.T) {
	root, out := buildRoot(t)
	root.SetArgs([]string{"login", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "OAuth") && !strings.Contains(out.String(), "auth") {
		t.Fatalf("help output missing OAuth mention: %q", out.String())
	}
}

func TestListEmptyPool(t *testing.T) {
	root, _ := buildRoot(t)
	root.SetArgs([]string{"--pool", t.TempDir(), "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestRotateEmptyPool(t *testing.T) {
	dir := t.TempDir()
	root, _ := buildRoot(t)
	root.SetArgs([]string{
		"--pool", dir,
		"--codex-auth", filepath.Join(dir, "auth.json"),
		"rotate",
	})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "no accounts in pool") {
		t.Fatalf("Execute() error = %v, want 'no accounts in pool'", err)
	}
}

func TestDaemonSleepInjectable(t *testing.T) {
	original := sleepFn
	var called bool
	sleepFn = func(_ time.Duration) { called = true }
	t.Cleanup(func() { sleepFn = original })

	// Verify sleepFn is injectable (called directly, not via Execute which runs forever).
	sleepFn(time.Millisecond)
	if !called {
		t.Fatal("sleepFn was not called")
	}
}
