package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func GeneratePKCECodes() (*PKCECodes, error) {
	verifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}
	return &PKCECodes{
		CodeVerifier:  verifier,
		CodeChallenge: generateCodeChallenge(verifier),
	}, nil
}

func generateCodeVerifier() (string, error) {
	b := make([]byte, 96)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
}

func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(h[:])
}
