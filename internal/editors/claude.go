// claude.go writes Claude Code configuration so the agent uses swytchcode exec (init-time only).
package editors

// WriteClaudeConfig will emit Claude Code configuration (or equivalent)
// that ensures the agent uses `swytchcode exec` as the single execution
// path for tools.
//
// Like the other editor writers, this is invoked only during `swytchcode init`
// and must never be consulted at runtime by the kernel.
func WriteClaudeConfig(projectRoot string) error {
	// TODO: Implement Claude editor config emission.
	_ = projectRoot
	return nil
}

