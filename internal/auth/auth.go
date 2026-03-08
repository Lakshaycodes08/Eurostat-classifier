// auth.go manages CLI authentication: session file storage and token resolution.
// Two auth modes are supported (checked in order):
//  1. Service token: SWYTCHCODE_TOKEN env var (agents, CI/CD)
//  2. User session:  ~/.swytchcode/auth.json written by `swytchcode login`
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gitlab.com/swytchcode/shell/internal/constants"
)

// AuthSession is the credential stored in ~/.swytchcode/auth.json after `swytchcode login`.
type AuthSession struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	CustomerUUID string `json:"customer_uuid"`
	Email        string `json:"email"`
	ExpiresAt    int64  `json:"expires_at"` // Unix timestamp
}

// SessionPath returns the absolute path to the session file.
func SessionPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ".swytchcode", "auth.json"), nil
}

// Load reads and parses the session file. Returns an error if the file is missing
// or malformed; does NOT check expiry — callers handle that.
func Load() (*AuthSession, error) {
	path, err := SessionPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not logged in")
		}
		return nil, fmt.Errorf("read session file: %w", err)
	}
	var s AuthSession
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse session file: %w", err)
	}
	return &s, nil
}

// Save writes the session to disk with mode 0600 (owner-only).
func Save(s *AuthSession) error {
	path, err := SessionPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create .swytchcode dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// Delete removes the session file. Returns nil if the file does not exist.
func Delete() error {
	path, err := SessionPath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove session file: %w", err)
	}
	return nil
}

// RefreshIfExpired auto-refreshes s if it is expired and saves the updated session.
// Returns nil immediately if the session is still valid.
// Returns an error if expired with no refresh token, or if the refresh/save fails.
func RefreshIfExpired(s *AuthSession, apiURL string) error {
	if !s.IsExpired() {
		return nil
	}
	if s.RefreshToken == "" {
		return fmt.Errorf("session expired — run `swytchcode login`")
	}
	if err := s.Refresh(apiURL); err != nil {
		return err
	}
	return Save(s)
}

// IsExpired reports whether the session token has expired (with a safety buffer).
func (s *AuthSession) IsExpired() bool {
	return time.Now().Unix() >= s.ExpiresAt-constants.SessionRefreshBufferSecs
}

// Refresh calls POST /v2/cli/auth/refresh, updates the session in place, and persists it.
// On auth failure (400/401) it deletes the session file so the user must re-login.
func (s *AuthSession) Refresh(apiURL string) error {
	body, _ := json.Marshal(map[string]string{"refresh_token": s.RefreshToken})
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		apiURL+"/v2/cli/auth/refresh", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := constants.NewHTTPClient(constants.AuthRequestTimeout)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("token refresh failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest {
		_ = Delete()
		return fmt.Errorf("session expired — run `swytchcode login` again")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refresh returned %d", resp.StatusCode)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"` // seconds
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode refresh response: %w", err)
	}
	s.AccessToken = result.AccessToken
	if result.RefreshToken != "" {
		s.RefreshToken = result.RefreshToken
	}
	if result.ExpiresIn > 0 {
		s.ExpiresAt = time.Now().Unix() + int64(result.ExpiresIn)
	}
	return nil
}

// ResolveToken returns a bearer token for API calls, trying in order:
//  1. SWYTCHCODE_TOKEN env var (service token — agents, CI)
//  2. ~/.swytchcode/auth.json (user session from `swytchcode login`)
//
// The second return value is true if the token came from the session file
// (i.e. it is a Firebase JWT, not a service token).
func ResolveToken() (token string, fromSession bool, err error) {
	if t := os.Getenv("SWYTCHCODE_TOKEN"); t != "" {
		return t, false, nil
	}
	s, err := Load()
	if err != nil {
		return "", false, fmt.Errorf("not logged in — run `swytchcode login` or set SWYTCHCODE_TOKEN")
	}
	if s.IsExpired() {
		if s.RefreshToken == "" {
			return "", false, fmt.Errorf("session expired — run `swytchcode login`")
		}
		apiURL := os.Getenv("SWYTCHCODE_API_URL")
		if apiURL == "" {
			apiURL = "https://api-v2.swytchcode.com"
		}
		if err := s.Refresh(apiURL); err != nil {
			return "", false, err
		}
		if err := Save(s); err != nil {
			return "", false, fmt.Errorf("save refreshed session: %w", err)
		}
	}
	return s.AccessToken, true, nil
}
