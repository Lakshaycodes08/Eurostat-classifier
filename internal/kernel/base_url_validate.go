package kernel

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateExecutionBaseURL rejects non-HTTPS base URLs except http on loopback hosts.
func ValidateExecutionBaseURL(baseURL string) error {
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid base URL: %q", baseURL)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme == "https" {
		return nil
	}
	if scheme != "http" {
		return fmt.Errorf("integration endpoint must use HTTPS (or http on localhost only): %q", baseURL)
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return nil
	}
	return fmt.Errorf("integration endpoint must use HTTPS except for localhost loopback: %q", baseURL)
}
