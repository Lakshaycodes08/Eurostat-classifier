// spinner.go provides a simple spinner for showing progress during long operations.
package util

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Spinner provides a simple terminal spinner.
type Spinner struct {
	message string
	done    chan bool
	writer  io.Writer
	stopped bool
}

// NewSpinner creates a new spinner with the given message.
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		done:    make(chan bool, 1),
		writer:  os.Stderr,
		stopped: false,
	}
}

// Start starts the spinner animation.
func (s *Spinner) Start() {
	if s.stopped {
		return
	}
	go func() {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				if s.stopped {
					return
				}
				fmt.Fprintf(s.writer, "\r%s %s", frames[i%len(frames)], s.message)
				i++
			}
		}
	}()
}

// Stop stops the spinner and clears the line.
func (s *Spinner) Stop() {
	if s.stopped {
		return
	}
	s.stopped = true
	select {
	case s.done <- true:
	default:
	}
	// Small delay to ensure the goroutine sees the done signal
	time.Sleep(120 * time.Millisecond)
	fmt.Fprintf(s.writer, "\r\033[K") // Clear line
}

// StopWithMessage stops the spinner and prints a message.
func (s *Spinner) StopWithMessage(message string) {
	if s.stopped {
		fmt.Fprintf(s.writer, "%s\n", message)
		return
	}
	s.stopped = true
	select {
	case s.done <- true:
	default:
	}
	// Small delay to ensure the goroutine sees the done signal
	time.Sleep(120 * time.Millisecond)
	fmt.Fprintf(s.writer, "\r\033[K%s\n", message) // Clear line and print message
}
