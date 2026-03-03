// auth.go manages CLI authentication: session file storage and token resolution.
// Two auth modes are supported (checked in order):
//  1. Service token: SWYTCHCODE_TOKEN env var (agents, CI/CD)
//  2. User session:  ~/.swytchcode/auth.json written by `swytchcode login`
package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AuthSession is the credential stored in ~/.swytchcode/auth.json after `swytchcode login`.
type AuthSession struct {
	AccessToken  string `json:"access_token"`
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

// IsExpired reports whether the session token has expired.
func (s *AuthSession) IsExpired() bool {
	return time.Now().Unix() >= s.ExpiresAt
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
		return "", false, fmt.Errorf("session expired — run `swytchcode login`")
	}
	return s.AccessToken, true, nil
}
