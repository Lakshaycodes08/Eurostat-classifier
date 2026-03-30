package registry

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

var insecureWarn sync.Once

func warnInsecureTLS() {
	insecureWarn.Do(func() {
		_, _ = fmt.Fprintf(os.Stderr, "WARNING: TLS verification is disabled (SWYTCHCODE_INSECURE=1). Do not use in production or CI.\n")
	})
}

func envTruthy(name string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes"
}

// checkInsecureBlockedInCI returns an error when SWYTCHCODE_INSECURE is set in a CI environment.
func checkInsecureBlockedInCI() error {
	if os.Getenv("SWYTCHCODE_INSECURE") != "1" {
		return nil
	}
	if envTruthy("CI") || envTruthy("GITHUB_ACTIONS") || envTruthy("GITLAB_CI") {
		return fmt.Errorf("SWYTCHCODE_INSECURE=1 cannot be used in CI (CI, GITHUB_ACTIONS, or GITLAB_CI is set)")
	}
	warnInsecureTLS()
	return nil
}

// IsCILikeEnv reports whether the environment looks like a CI job (used by doctor and diagnostics).
func IsCILikeEnv() bool {
	return envTruthy("CI") || envTruthy("GITHUB_ACTIONS") || envTruthy("GITLAB_CI")
}
