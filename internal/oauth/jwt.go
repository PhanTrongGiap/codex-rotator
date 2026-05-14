package oauth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type jwtClaims struct {
	Email     string    `json:"email"`
	CodexAuth codexAuth `json:"https://api.openai.com/auth"`
}

type codexAuth struct {
	ChatgptAccountID string `json:"chatgpt_account_id"`
	ChatgptPlanType  string `json:"chatgpt_plan_type"`
}

func parseJWT(token string) (*jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT: expected 3 parts")
	}
	data, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, err
	}
	var c jwtClaims
	if err = json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func base64URLDecode(s string) ([]byte, error) {
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}
