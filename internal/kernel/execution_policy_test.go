package kernel

import (
	"testing"

	"gitlab.com/swytchcode/swytchcode-cli/internal/manifest"
)

func TestResolveExecutionPolicy_Defaults(t *testing.T) {
	r := ResolveExecutionPolicy(nil)
	if r.MaxRetries != 3 {
		t.Fatalf("MaxRetries = %d", r.MaxRetries)
	}
	if r.IdempotencyMode != "none" {
		t.Fatalf("IdempotencyMode = %q", r.IdempotencyMode)
	}
}

func TestResolveExecutionPolicy_Overrides(t *testing.T) {
	mr := 1
	bd := 100
	to := 5000
	mode := "stripe_style"
	hdr := "Idempotency-Key"
	e := &manifest.Entry{
		ExecutionPolicy: &manifest.ExecutionPolicy{
			MaxRetries:    &mr,
			BaseDelayMs:   &bd,
			HTTPTimeoutMs: &to,
			Idempotency: &manifest.IdempotencyPolicy{
				Mode:       mode,
				HeaderName: hdr,
			},
		},
	}
	r := ResolveExecutionPolicy(e)
	if r.MaxRetries != 1 || r.IdempotencyMode != "stripe_style" {
		t.Fatalf("%+v", r)
	}
}
