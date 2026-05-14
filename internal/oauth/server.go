package oauth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

const successHTML = `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>Auth OK</title></head>
<body style="font-family:sans-serif;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#f0fdf4">
<div style="text-align:center"><h1 style="color:#16a34a">&#10003; Authentication successful</h1>
<p>You can close this tab and return to the terminal.</p></div></body></html>`

// OAuthServer is a local HTTP server that captures the OAuth callback.
type OAuthServer struct {
	port       int
	server     *http.Server
	resultChan chan *OAuthResult
	errorChan  chan error
	mu         sync.Mutex
	running    bool
}

func NewOAuthServer(port int) *OAuthServer {
	return &OAuthServer{
		port:       port,
		resultChan: make(chan *OAuthResult, 1),
		errorChan:  make(chan error, 1),
	}
}

func (s *OAuthServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return fmt.Errorf("server already running")
	}
	if !s.portAvailable() {
		return fmt.Errorf("port %d already in use", s.port)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/callback", s.handleCallback)
	mux.HandleFunc("/success", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(successHTML))
	})
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	s.running = true
	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.errorChan <- err
		}
	}()
	time.Sleep(80 * time.Millisecond)
	return nil
}

func (s *OAuthServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running || s.server == nil {
		return nil
	}
	err := s.server.Shutdown(ctx)
	s.running = false
	s.server = nil
	return err
}

func (s *OAuthServer) WaitForCallback(timeout time.Duration) (*OAuthResult, error) {
	select {
	case r := <-s.resultChan:
		return r, nil
	case err := <-s.errorChan:
		return nil, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for OAuth callback")
	}
}

func (s *OAuthServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if errParam := q.Get("error"); errParam != "" {
		s.send(&OAuthResult{Error: errParam})
		http.Error(w, "OAuth error: "+errParam, http.StatusBadRequest)
		return
	}
	code, state := q.Get("code"), q.Get("state")
	if code == "" || state == "" {
		s.send(&OAuthResult{Error: "missing_params"})
		http.Error(w, "missing code or state", http.StatusBadRequest)
		return
	}
	s.send(&OAuthResult{Code: code, State: state})
	http.Redirect(w, r, "/success", http.StatusFound)
}

func (s *OAuthServer) send(r *OAuthResult) {
	select {
	case s.resultChan <- r:
	default:
	}
}

func (s *OAuthServer) portAvailable() bool {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}
