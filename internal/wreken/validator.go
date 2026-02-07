// validator.go will validate Wrekenfile structure and schemas so invalid files fail fast.
package wreken

// This file will contain validation logic for:
//
//   - Wrekenfile structure itself (required fields, shapes).
//   - Per-tool argument schemas.
//   - Declared environment variables and allowed operations.
//
// The intent is to fail fast and deterministically when Wrekenfiles
// are invalid, rather than attempting "best effort" execution.

