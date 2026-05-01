package schedulesdirect

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
)


// Account is the credential body for POST /token.
//
// Password must be the lowercase sha1-hex of the user's plaintext password
// (40 characters; pattern ^[a-f0-9]{40}$). Use HashPassword to compute it.
//
// NewToken, when true, instructs SD to invalidate the existing valid token (if
// any) and issue a fresh one. Default false: SD returns the existing token
// when one is still within its 24h validity window. There is a limited number
// of token requests per 24h period; consumers should leave NewToken false
// unless they have specific reason to force a new token.
type Account struct {
	Username string `json:"username"`
	Password string `json:"password"`
	NewToken bool   `json:"newToken,omitempty"`
}

// HashPassword returns the lowercase sha1-hex of plaintext, the format SD
// requires for the Account.Password field.
func HashPassword(plaintext string) string {
	sum := sha1.Sum([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// Token is a successful response from POST /token.
//
// TokenExpires is SD's authoritative expiry as a UNIX epoch timestamp;
// consumers compute their own refresh margin against it. ServerTime carries
// SD's clock at response time, useful for clock-skew calibration.
type Token struct {
	BaseResponse
	Token        string `json:"token"`
	TokenExpires int64  `json:"tokenExpires,omitempty"`
	ServerTime   int64  `json:"serverTime,omitempty"`
}

// GetToken POSTs to /token and returns a Token on success.
//
// On a non-zero envelope code (including SERVICE_OFFLINE / 3000 with its
// placeholder token), GetToken returns an *Error and does not return a Token.
// On transport / decode failures, the returned error is wrapped with
// fmt.Errorf and is not an *Error.
func (c *Client) GetToken(ctx context.Context, account Account) (*Token, error) {
	body, err := json.Marshal(account)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: marshal account: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"token", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: http: %w", err)
	}
	var tok Token
	if err := c.readResponse(resp, &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}
