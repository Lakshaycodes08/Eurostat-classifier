// config.go provides registry configuration using build-time constants.
package registry

import (
	"os"

	"gitlab.com/swytchcode/cli/internal/constants"
)

// Config holds the registry API configuration.
type Config struct {
	BaseURL string
}

// DefaultConfig returns the default registry configuration.
// It respects the SWYTCHCODE_API_URL environment variable, falling back to the build-time constant.
func DefaultConfig() *Config {
	u := os.Getenv("SWYTCHCODE_API_URL")
	if u == "" {
		u = constants.RegistryURL
	}
	return &Config{BaseURL: u}
}

// APIBasePath returns the full API base path with version prefix.
func (c *Config) APIBasePath() string {
	return c.BaseURL + "/v2/cli"
}
