// vscode.go writes VS Code configuration so the editor uses swytchcode exec (init-time only).
package editors

// WriteVSCodeConfig will emit VS Code / VS Code plugin configuration
// that points any relevant commands or tasks at `swytchcode exec`.
//
// The exact shape is left as a TODO until the extension and task model
// is fully defined. This function must not be used at runtime by the
// kernel; it is strictly init-time behavior.
func WriteVSCodeConfig(projectRoot string) error {
	// TODO: Implement VS Code config emission.
	_ = projectRoot
	return nil
}

