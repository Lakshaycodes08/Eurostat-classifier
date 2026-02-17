// config.go provides registry configuration using build-time constants.
package registry

import "gitlab.com/swytchcode/shell/internal/constants"

// Config holds the registry API configuration.
type Config struct {
	BaseURL string
}

// DefaultConfig returns the default registry configuration using build-time constants.
func DefaultConfig() *Config {
	return &Config{BaseURL: constants.RegistryURL}
}

// ConfigFromProjectRoot returns registry config using build-time constants.
// The registry URL is set at build time and does not vary by project or environment.
func ConfigFromProjectRoot(projectRoot string) *Config {
	return DefaultConfig()
}

// APIBasePath returns the full API base path with version prefix.
func (c *Config) APIBasePath() string {
	return c.BaseURL + "/v2/shell"
}
