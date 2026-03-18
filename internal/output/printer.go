// printer.go provides shared output helpers for consistent CLI message formatting.
package output

import (
	"fmt"
	"io"
)

// Error prints "✗ msg" to w (use os.Stderr).
func Error(w io.Writer, msg string) { fmt.Fprintf(w, "✗ %s\n", msg) }

// Warn prints "⚠ msg" to w (use os.Stderr).
func Warn(w io.Writer, msg string) { fmt.Fprintf(w, "⚠ %s\n", msg) }

// Success prints "✓ msg" to w (use os.Stdout).
func Success(w io.Writer, msg string) { fmt.Fprintf(w, "✓ %s\n", msg) }

// Hint prints an indented hint line to w (use os.Stderr).
func Hint(w io.Writer, msg string) { fmt.Fprintf(w, "  hint: %s\n", msg) }

// ValidationErrors prints all errors under a single header to w (use os.Stderr).
func ValidationErrors(w io.Writer, errs []string) {
	fmt.Fprintln(w, "✗ Validation errors:")
	for _, e := range errs {
		fmt.Fprintf(w, "  • %s\n", e)
	}
}
