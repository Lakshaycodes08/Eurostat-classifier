// login.go implements the device-flow OAuth login for swytchcode login.
package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gitlab.com/swytchcode/shell/internal/auth"
	"gitlab.com/swytchcode/shell/internal/constants"
)

// LoginConfig holds configuration for the login flow.
type LoginConfig struct {
	APIURL string          // base URL of the backend API
	OnURL  func(url string) // called with the verification URL before polling starts (optional)
}

// startResponse is returned by POST /v2/cli/auth/start.
type startResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`       // seconds
	PollIntervalMs  int    `json:"poll_interval_ms"` // milliseconds between polls
}

// pollResponse is returned by POST /v2/cli/auth/poll.
type pollResponse struct {
	Status       string `json:"status"`        // "pending" | "expired" | "verified"
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"` // nullable until web app sends it
	CustomerUUID string `json:"customer_uuid"`
	Email        string `json:"email"`
}

const maxPollAttempts = 200

// RunLogin executes the device-flow login. It prints progress to w and saves
// the session to ~/.swytchcode/auth.json on success.
func RunLogin(cfg LoginConfig, w io.Writer) error {
	client := &http.Client{Timeout: constants.HTTPClientTimeout}

	// Step 1: start device flow
	startResp, err := startDeviceFlow(client, cfg.APIURL)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "Visit the URL below to log in:\n\n  %s\n\n", startResp.VerificationURL)
	if cfg.OnURL != nil {
		cfg.OnURL(startResp.VerificationURL)
	}
	fmt.Fprintf(w, "Waiting for browser authentication...\n")

	pollInterval := time.Duration(startResp.PollIntervalMs) * time.Millisecond
	if pollInterval == 0 {
		pollInterval = 3 * time.Second
	}

	// Step 2: poll until verified or expired
	for attempt := 0; attempt < maxPollAttempts; attempt++ {
		time.Sleep(pollInterval)

		result, err := pollDeviceFlow(client, cfg.APIURL, startResp.DeviceCode)
		if err != nil {
			return err
		}

		switch result.Status {
		case "verified":
			session := &auth.AuthSession{
				AccessToken:  result.AccessToken,
				RefreshToken: result.RefreshToken,
				CustomerUUID: result.CustomerUUID,
				Email:        result.Email,
				ExpiresAt:    time.Now().Unix() + 3600 - 60, // 55-minute safety buffer
			}
			if err := auth.Save(session); err != nil {
				return fmt.Errorf("save session: %w", err)
			}
			fmt.Fprintf(w, "Logged in as %s\n", result.Email)
			return nil

		case "expired":
			return fmt.Errorf("timed out — run `swytchcode login` again")

		default: // "pending"
			continue
		}
	}

	return fmt.Errorf("timed out after %d attempts — run `swytchcode login` again", maxPollAttempts)
}

func startDeviceFlow(client *http.Client, apiURL string) (*startResponse, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		apiURL+"/v2/cli/auth/start", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build start request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("start request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("start returned %d", resp.StatusCode)
	}

	var result startResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode start response: %w", err)
	}
	return &result, nil
}

func pollDeviceFlow(client *http.Client, apiURL, deviceCode string) (*pollResponse, error) {
	body, _ := json.Marshal(map[string]string{"device_code": deviceCode})
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		apiURL+"/v2/cli/auth/poll", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build poll request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("poll request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("poll returned %d", resp.StatusCode)
	}

	var result pollResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode poll response: %w", err)
	}
	return &result, nil
}
