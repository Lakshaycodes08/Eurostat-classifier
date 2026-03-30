// execution_policy.go resolves manifest execution_policy with kernel defaults.
package kernel

import (
	"fmt"
	"strings"
	"time"

	"gitlab.com/swytchcode/swytchcode-cli/internal/constants"
	"gitlab.com/swytchcode/swytchcode-cli/internal/manifest"
)

// ResolvedExecutionPolicy is the effective HTTP policy for one integration after merging manifest + defaults.
type ResolvedExecutionPolicy struct {
	MaxRetries        int
	BaseDelay         time.Duration
	HTTPTimeout       time.Duration
	IdempotencyMode   string
	IdempotencyHeader string
}

// DefaultResolvedExecutionPolicy returns built-in defaults when manifest omits execution_policy.
func DefaultResolvedExecutionPolicy() ResolvedExecutionPolicy {
	return ResolvedExecutionPolicy{
		MaxRetries:        3,
		BaseDelay:         500 * time.Millisecond,
		HTTPTimeout:       constants.HTTPClientTimeout,
		IdempotencyMode:   "none",
		IdempotencyHeader: "Idempotency-Key",
	}
}

// ResolveExecutionPolicy merges manifest.ExecutionPolicy into defaults.
func ResolveExecutionPolicy(entry *manifest.Entry) ResolvedExecutionPolicy {
	r := DefaultResolvedExecutionPolicy()
	if entry == nil || entry.ExecutionPolicy == nil {
		return r
	}
	p := entry.ExecutionPolicy
	if p.MaxRetries != nil {
		r.MaxRetries = *p.MaxRetries
		if r.MaxRetries < 0 {
			r.MaxRetries = 0
		}
	}
	if p.BaseDelayMs != nil && *p.BaseDelayMs >= 0 {
		r.BaseDelay = time.Duration(*p.BaseDelayMs) * time.Millisecond
	}
	if p.HTTPTimeoutMs != nil && *p.HTTPTimeoutMs > 0 {
		r.HTTPTimeout = time.Duration(*p.HTTPTimeoutMs) * time.Millisecond
	}
	if p.Idempotency != nil {
		mode := strings.TrimSpace(strings.ToLower(p.Idempotency.Mode))
		if mode != "" {
			r.IdempotencyMode = mode
		}
		if h := strings.TrimSpace(p.Idempotency.HeaderName); h != "" {
			r.IdempotencyHeader = h
		}
	}
	return r
}

// ManifestProjectLibrary returns the manifest map key for tool.Integration ("project.library@version").
func ManifestProjectLibrary(integration string) (string, error) {
	parts := strings.Split(integration, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid integration format: %q", integration)
	}
	return strings.TrimSpace(parts[0]), nil
}

// GetExecutionPolicy loads manifest and returns resolved policy for the integration.
func GetExecutionPolicy(projectRoot, integration string) (ResolvedExecutionPolicy, error) {
	key, err := ManifestProjectLibrary(integration)
	if err != nil {
		return ResolvedExecutionPolicy{}, err
	}
	m, err := manifest.Read(projectRoot)
	if err != nil {
		return ResolvedExecutionPolicy{}, fmt.Errorf("read manifest: %w", err)
	}
	ent, ok := m[key]
	if !ok {
		return DefaultResolvedExecutionPolicy(), nil
	}
	return ResolveExecutionPolicy(&ent), nil
}
