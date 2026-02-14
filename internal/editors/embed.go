package editors

import "embed"

// templates holds editor rule templates copied into the project on swytchcode init.
// Keep in sync with top-level editors/ for human editing; these are embedded in the binary.
//go:embed all:templates
var templates embed.FS
