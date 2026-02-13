// config.go loads registry base URL from tooling.json and env (env wins; never persisted back to disk).
package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the registry API configuration.
type Config struct {
	BaseURL string
}

// DefaultConfig returns the default registry configuration.
// Base URL can be overridden via SWYTCHCODE_REGISTRY_URL environment variable.
func DefaultConfig() *Config {
	return &Config{BaseURL: baseURLFromEnvOrDefault("")}
}

// baseURLFromEnvOrDefault returns BaseURL: env wins, else fallback, else "https://localhost".
func baseURLFromEnvOrDefault(fallback string) string {
	if u := os.Getenv("SWYTCHCODE_REGISTRY_URL"); u != "" {
		return u
	}
	if fallback != "" {
		return fallback
	}
	return "https://localhost"
}

// ConfigFromProjectRoot loads registry config from .swytchcode/tooling.json when present.
// registry_url in tooling.json is used unless SWYTCHCODE_REGISTRY_URL is set (env wins).
// If tooling.json is missing or has no registry_url, returns DefaultConfig().
func ConfigFromProjectRoot(projectRoot string) *Config {
	effective, _ := RegistryURLEffectiveAndSource(projectRoot)
	return &Config{BaseURL: effective}
}

// RegistryURLEffectiveAndSource returns the effective registry base URL and its source.
// Source is "env" when SWYTCHCODE_REGISTRY_URL is set, otherwise "tooling" (or "default" if no tooling.json).
// Used by swytchcode config so overrides are visible and never silently persisted.
func RegistryURLEffectiveAndSource(projectRoot string) (effectiveURL, source string) {
	fromEnv := os.Getenv("SWYTCHCODE_REGISTRY_URL")
	if fromEnv != "" {
		return fromEnv, "env"
	}
	toolingPath := filepath.Join(projectRoot, ".swytchcode", "tooling.json")
	data, err := os.ReadFile(toolingPath)
	if err != nil {
		return "https://localhost", "default"
	}
	var tooling struct {
		RegistryURL string `json:"registry_url"`
	}
	if err := json.Unmarshal(data, &tooling); err != nil || tooling.RegistryURL == "" {
		return "https://localhost", "default"
	}
	return tooling.RegistryURL, "tooling"
}

// APIBasePath returns the full API base path with version prefix.
func (c *Config) APIBasePath() string {
	return c.BaseURL + "/v2/shell"
}
