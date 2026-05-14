package main

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/router-for-me/codex-rotator/internal/oauth"
	"github.com/router-for-me/codex-rotator/internal/probe"
	"github.com/router-for-me/codex-rotator/internal/rotate"
	"github.com/router-for-me/codex-rotator/internal/store"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	flagPool      string
	flagCodexAuth string
)

func defaultCodexAuth() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex", "auth.json")
}

func main() {
	root := &cobra.Command{
		Use:   "codex-rotator",
		Short: "Manage & auto-rotate Codex ChatGPT accounts",
	}

	root.PersistentFlags().StringVar(&flagPool, "pool", store.DefaultPoolDir(), "Pool directory for codex accounts")
	root.PersistentFlags().StringVar(&flagCodexAuth, "codex-auth", defaultCodexAuth(), "Path to /root/.codex/auth.json")

	root.AddCommand(
		cmdLogin(),
		cmdList(),
		cmdRotate(),
		cmdDaemon(),
		cmdRun(),
		cmdImport(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// ── login ─────────────────────────────────────────────────────────────────────

func cmdLogin() *cobra.Command {
	var openBrowserFlag bool
	var callbackURL string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Add a new Codex account via OAuth",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return doLogin(openBrowserFlag, callbackURL)
		},
	}
	cmd.Flags().BoolVar(&openBrowserFlag, "open-browser", false, "Open the auth URL with a local browser")
	cmd.Flags().StringVar(&callbackURL, "callback-url", "", "Skip browser flow; provide the OAuth callback URL directly")
	cmd.Flags().Bool("no-browser", true, "Deprecated: login prints the auth URL by default")
	_ = cmd.Flags().MarkHidden("no-browser")
	return cmd
}

func doLogin(openBrowserFlag bool, callbackURL string) error {
	pkce, err := oauth.GeneratePKCECodes()
	if err != nil {
		return fmt.Errorf("pkce: %w", err)
	}
	state, err := oauth.GenerateState()
	if err != nil {
		return fmt.Errorf("state: %w", err)
	}

	var result *oauth.OAuthResult

	if callbackURL != "" {
		// Non-interactive mode: caller provides the callback URL directly.
		result, err = parseCallbackInput(callbackURL)
		if err != nil {
			return fmt.Errorf("callback-url: %w", err)
		}
	} else {
		srv := oauth.NewOAuthServer(1455)
		if err = srv.Start(); err != nil {
			return fmt.Errorf("start oauth server: %w", err)
		}
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = srv.Stop(ctx)
		}()

		authURL := oauth.GenerateAuthURL(state, pkce)

		if openBrowserFlag {
			fmt.Println("Opening browser for Codex authentication…")
			if err = openBrowser(authURL); err != nil {
				fmt.Printf("Could not open browser automatically: %v\n", err)
			}
		}
		fmt.Println("Open the following URL in your browser:")
		fmt.Println(authURL)
		fmt.Println("Waiting for OAuth callback (5 min timeout)…")
		fmt.Println("  On a remote server? Paste the full callback URL below and press Enter.")
		fmt.Println("  Note: callback URLs are one-time use — do not share them.")
		fmt.Println("  Paste callback URL (or press Ctrl+C to cancel):")

		result, err = waitForCallbackOrPaste(srv, 5*time.Minute)
		if err != nil {
			return fmt.Errorf("callback: %w", err)
		}
	}

	if result.Error != "" {
		return fmt.Errorf("oauth error: %s", result.Error)
	}
	if callbackURL == "" && result.State != state {
		return fmt.Errorf("state mismatch — possible CSRF")
	}

	td, err := oauth.ExchangeCodeForTokens(context.Background(), result.Code, pkce)
	if err != nil {
		return fmt.Errorf("token exchange: %w", err)
	}

	planType, hashID := oauth.PlanTypeAndHash(td.IDToken)
	fileName := store.CredentialFileName(td.Email, planType, hashID)

	if err = os.MkdirAll(flagPool, 0700); err != nil {
		return fmt.Errorf("creating pool dir: %w", err)
	}
	t := &store.CodexToken{
		IDToken:      td.IDToken,
		AccessToken:  td.AccessToken,
		RefreshToken: td.RefreshToken,
		AccountID:    td.AccountID,
		Email:        td.Email,
		Expire:       td.Expire,
		LastRefresh:  time.Now().UTC().Format(time.RFC3339),
	}
	dest := filepath.Join(flagPool, fileName)
	if err = t.Save(dest); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}
	fmt.Printf("✓ Saved: %s\n", dest)
	return nil
}

type callbackWaitResult struct {
	result *oauth.OAuthResult
	err    error
}

func waitForCallbackOrPaste(srv *oauth.OAuthServer, timeout time.Duration) (*oauth.OAuthResult, error) {
	callbackCh := make(chan callbackWaitResult, 1)
	go func() {
		r, err := srv.WaitForCallback(timeout)
		callbackCh <- callbackWaitResult{result: r, err: err}
	}()

	pasteCh := make(chan callbackWaitResult, 1)
	go readCallbackURLFromStdin(pasteCh)

	select {
	case r := <-callbackCh:
		return r.result, r.err
	case r := <-pasteCh:
		return r.result, r.err
	}
}

func readCallbackURLFromStdin(out chan<- callbackWaitResult) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		result, err := parseCallbackInput(raw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid callback URL: %v\nPaste the full callback URL, or keep waiting for the browser callback.\n", err)
			continue
		}
		out <- callbackWaitResult{result: result}
		return
	}
	if err := scanner.Err(); err != nil {
		out <- callbackWaitResult{err: err}
	}
}

func parseCallbackInput(raw string) (*oauth.OAuthResult, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if q.Get("code") == "" && strings.Contains(raw, "code=") {
		q, err = url.ParseQuery(strings.TrimPrefix(raw, "?"))
		if err != nil {
			return nil, err
		}
	}
	if errParam := q.Get("error"); errParam != "" {
		return &oauth.OAuthResult{Error: errParam}, nil
	}
	code, state := q.Get("code"), q.Get("state")
	if code == "" || state == "" {
		return nil, fmt.Errorf("missing code or state")
	}
	return &oauth.OAuthResult{Code: code, State: state}, nil
}

// ── list ──────────────────────────────────────────────────────────────────────

func cmdList() *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List accounts in the pool",
		RunE: func(_ *cobra.Command, _ []string) error {
			tokens, err := store.ListPool(flagPool)
			if err != nil {
				return err
			}
			if len(tokens) == 0 {
				fmt.Printf("No accounts in pool %s\n", flagPool)
				return nil
			}
			fmt.Printf("Pool: %s (%d accounts)\n\n", flagPool, len(tokens))
			for _, t := range tokens {
				status := "(not checked)"
				if check {
					s := probe.CheckToken(t.AccessToken)
					status = s.String()
				}
				fmt.Printf("  %-40s  %s\n", t.Email, status)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "Probe each token (slower)")
	return cmd
}

// ── rotate ────────────────────────────────────────────────────────────────────

func cmdRotate() *cobra.Command {
	return &cobra.Command{
		Use:   "rotate",
		Short: "Check current token; swap to a working account if needed",
		RunE: func(_ *cobra.Command, _ []string) error {
			rotated, email, err := rotate.Once(flagCodexAuth, flagPool)
			if err != nil {
				return err
			}
			if rotated {
				fmt.Printf("✓ Rotated to: %s\n", email)
			} else {
				fmt.Println("✓ Current token still valid — no rotation needed")
			}
			return nil
		},
	}
}

// ── daemon ────────────────────────────────────────────────────────────────────

// sleepFn is injectable for tests to avoid real sleeps.
var sleepFn = time.Sleep

func cmdDaemon() *cobra.Command {
	var interval time.Duration
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run rotate in a loop at a fixed interval",
		RunE: func(_ *cobra.Command, _ []string) error {
			log.Infof("daemon started (interval=%s, pool=%s)", interval, flagPool)
			for {
				rotated, email, err := rotate.Once(flagCodexAuth, flagPool)
				if err != nil {
					log.Warnf("rotate error: %v", err)
				} else if rotated {
					log.Infof("rotated to %s", email)
				} else {
					log.Info("token valid, no rotation")
				}
				sleepFn(interval)
			}
		},
	}
	cmd.Flags().DurationVar(&interval, "interval", 60*time.Second, "Check interval (e.g. 60s, 2m)")
	return cmd
}

// ── run ───────────────────────────────────────────────────────────────────────

func cmdRun() *cobra.Command {
	return &cobra.Command{
		Use:                "run [-- codex args...]",
		Short:              "Rotate once then exec codex CLI",
		DisableFlagParsing: true,
		RunE: func(_ *cobra.Command, args []string) error {
			rotated, email, err := rotate.Once(flagCodexAuth, flagPool)
			if err != nil {
				fmt.Fprintf(os.Stderr, "rotate warning: %v\n", err)
			} else if rotated {
				fmt.Fprintf(os.Stderr, "rotated to: %s\n", email)
			}

			codexBin, err := exec.LookPath("codex")
			if err != nil {
				return fmt.Errorf("codex not found in PATH: %w", err)
			}

			// Strip leading "--" separator if present
			if len(args) > 0 && args[0] == "--" {
				args = args[1:]
			}

			return syscall.Exec(codexBin, append([]string{"codex"}, args...), os.Environ())
		},
	}
}

// ── import ────────────────────────────────────────────────────────────────────

func cmdImport() *cobra.Command {
	var fromDir string
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import accounts from a CLIProxyAPI auth directory",
		RunE: func(_ *cobra.Command, _ []string) error {
			return doImport(fromDir, flagPool)
		},
	}
	cmd.Flags().StringVar(&fromDir, "from", "", "Source directory containing credential JSON files (required)")
	_ = cmd.MarkFlagRequired("from")
	return cmd
}

func doImport(fromDir, poolDir string) error {
	entries, err := os.ReadDir(fromDir)
	if err != nil {
		return fmt.Errorf("reading source dir: %w", err)
	}
	if err = os.MkdirAll(poolDir, 0700); err != nil {
		return fmt.Errorf("creating pool dir: %w", err)
	}

	imported := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		src := filepath.Join(fromDir, e.Name())
		t, loadErr := store.Load(src)
		if loadErr != nil || t.AccessToken == "" {
			fmt.Fprintf(os.Stderr, "skipped %s: missing or empty access_token\n", e.Name())
			continue
		}
		destName := e.Name()
		if t.Email != "" {
			destName = store.CredentialFileName(t.Email, t.Type, t.AccountID)
		}
		data, err := os.ReadFile(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipped %s: %v\n", e.Name(), err)
			continue
		}
		dest := filepath.Join(poolDir, destName)
		if err = os.WriteFile(dest, data, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "skipped %s: %v\n", e.Name(), err)
			continue
		}
		fmt.Printf("imported: %s\n", dest)
		imported++
	}
	fmt.Printf("\n%d account(s) imported to %s\n", imported, poolDir)
	return nil
}

// openBrowser tries common commands to open a URL.
func openBrowser(url string) error {
	for _, cmd := range []string{"xdg-open", "open", "start"} {
		if path, err := exec.LookPath(cmd); err == nil {
			return exec.Command(path, url).Start()
		}
	}
	return fmt.Errorf("no browser launcher found")
}
