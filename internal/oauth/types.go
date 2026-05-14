package oauth

// PKCECodes holds PKCE code verifier and challenge.
type PKCECodes struct {
	CodeVerifier  string
	CodeChallenge string
}

// OAuthResult holds the result from the OAuth callback.
type OAuthResult struct {
	Code  string
	State string
	Error string
}

// TokenData holds OAuth tokens after exchange.
type TokenData struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    string `json:"account_id"`
	Email        string `json:"email"`
	Expire       string `json:"expired"`
}
