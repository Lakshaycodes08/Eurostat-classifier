// telemetry.go sends CLI execution events to the backend in a fire-and-forget goroutine.
// Failures are silently discarded — telemetry must never break the primary command.
package telemetry

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"

	"gitlab.com/swytchcode/cli/internal/auth"
	"gitlab.com/swytchcode/cli/internal/constants"
)

var hintOnce sync.Once

// MaybeHintNoAuth prints a one-time hint when telemetry is disabled due to no auth (no login, no SWYTCHCODE_TOKEN).
// Call when a command runs with token empty; do not call when SWYTCHCODE_TOKEN is set (silently skipped).
func MaybeHintNoAuth() {
	hintOnce.Do(func() {
		fmt.Fprintln(os.Stderr, "Telemetry is disabled. Run `swytchcode login` or set SWYTCHCODE_TOKEN to enable usage tracking.")
	})
}

// debugEnabled reports whether verbose telemetry logging is enabled.
func debugEnabled() bool {
	return os.Getenv(constants.EnvVarTelemetryDebug) != ""
}

// Event represents a single CLI execution event sent to POST /v2/cli/telemetry/batch.
type Event struct {
	EventID           string  `json:"event_id"`
	Command           string  `json:"command"`
	Outcome           string  `json:"outcome"` // "success" | "failure" | "cancelled"
	LibraryName       string  `json:"library_name,omitempty"`
	ProjectUUID       string  `json:"project_uuid,omitempty"`
	CLIVersion        string  `json:"cli_version"`
	ExecutionSource   string  `json:"execution_source,omitempty"`
	ErrorType         string  `json:"error_type,omitempty"` // only when outcome = "failure"
	DurationMs        *int64  `json:"duration_ms,omitempty"`
}

// APIError is an error that carries an HTTP status code for telemetry classification.
// Use NewAPIError(statusCode) so ClassifyError can return auth_error, rate_limit, or api_error.
type APIError struct {
	StatusCode int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d", e.StatusCode)
}

// NewAPIError returns an error that ClassifyError will classify by status code.
func NewAPIError(statusCode int) error {
	return &APIError{StatusCode: statusCode}
}

// EventOpts holds optional fields for building an event (duration, project UUID, pre-set error type).
type EventOpts struct {
	DurationMs  int64
	ProjectUUID string
	ErrorType   string // if set, used instead of ClassifyError(err)
}

// DetectExecutionSource returns execution_source for telemetry (mcp | ci | cli).
func DetectExecutionSource() string {
	if os.Getenv(constants.EnvVarTelemetryMCP) == "1" {
		return "mcp"
	}
	for _, v := range constants.EnvVarsCI {
		if os.Getenv(v) != "" {
			return "ci"
		}
	}
	return "cli"
}

// ClassifyError maps an error to a spec error_type (only when outcome = "failure").
func ClassifyError(err error) string {
	if err == nil {
		return ""
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == 401 || apiErr.StatusCode == 403 {
			return "auth_error"
		}
		if apiErr.StatusCode == 429 {
			return "rate_limit"
		}
		return "api_error"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	// Network errors: check for common patterns (url.Error wrapping net.Error)
	var netErr interface{ Timeout() bool; Temporary() bool }
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "timeout"
		}
		return "network_error"
	}
	return "unknown"
}

// newEventID generates a random UUID v4 string.
func newEventID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// Send fires off a telemetry event asynchronously when a token is set.
// When token is empty, no event is sent. fromSession is used only for auth refresh logic.
func Send(apiURL, token string, fromSession bool, ev Event) {
	if token == "" {
		if debugEnabled() {
			fmt.Fprintln(os.Stderr, "[telemetry] skipped (tokenEmpty=true)")
		}
		return
	}
	if ev.EventID == "" {
		ev.EventID = newEventID()
	}
	if ev.ExecutionSource == "" {
		ev.ExecutionSource = DetectExecutionSource()
	}
	if ev.CLIVersion == "" {
		ev.CLIVersion = constants.Version
	}
	if debugEnabled() {
		fmt.Fprintln(os.Stderr, "[telemetry] queue event",
			"command=", ev.Command,
			"outcome=", ev.Outcome,
			"library=", ev.LibraryName,
			"apiURL=", apiURL,
		)
	}
	go func() {
		if err := send(apiURL, token, fromSession, ev); err != nil && debugEnabled() {
			fmt.Fprintln(os.Stderr, "[telemetry] send error:", err.Error())
		}
	}()
}

// SendEvent builds and sends a command outcome event asynchronously. Only sends when token is set.
// err may be nil (outcome = "success") or non-nil (outcome = "failure"). opts may be nil.
func SendEvent(apiURL, token string, fromSession bool, command, library string, err error, opts *EventOpts) {
	if token == "" {
		if debugEnabled() {
			fmt.Fprintln(os.Stderr, "[telemetry] SendEvent skipped (tokenEmpty=true)")
		}
		return
	}
	ev := buildEvent(command, library, err, opts)
	Send(apiURL, token, fromSession, ev)
}

// SendEventSync builds and sends a command outcome event synchronously.
// Used for commands like exec where the process exits immediately after.
// err may be nil (outcome = "success") or non-nil (outcome = "failure"). opts may be nil.
func SendEventSync(apiURL, token string, fromSession bool, command, library string, err error, opts *EventOpts) {
	if token == "" {
		if debugEnabled() {
			fmt.Fprintln(os.Stderr, "[telemetry] SendEventSync skipped (tokenEmpty=true)")
		}
		return
	}
	ev := buildEvent(command, library, err, opts)
	if debugEnabled() {
		fmt.Fprintln(os.Stderr, "[telemetry] sync send event",
			"command=", ev.Command,
			"outcome=", ev.Outcome,
			"library=", ev.LibraryName,
			"apiURL=", apiURL,
		)
	}
	if err := send(apiURL, token, fromSession, ev); err != nil && debugEnabled() {
		fmt.Fprintln(os.Stderr, "[telemetry] sync send error:", err.Error())
	}
}

// buildEvent constructs an Event from the given inputs.
func buildEvent(command, library string, err error, opts *EventOpts) Event {
	outcome := "success"
	if err != nil {
		outcome = "failure"
	}
	ev := Event{
		Command:         command,
		LibraryName:     library,
		Outcome:         outcome,
		CLIVersion:      constants.Version,
		ExecutionSource: DetectExecutionSource(),
	}
	if opts != nil {
		if opts.DurationMs > 0 {
			ev.DurationMs = &opts.DurationMs
		}
		if opts.ProjectUUID != "" {
			ev.ProjectUUID = opts.ProjectUUID
		}
		if outcome == "failure" {
			if opts.ErrorType != "" {
				ev.ErrorType = opts.ErrorType
			} else {
				ev.ErrorType = ClassifyError(err)
			}
		}
	} else if outcome == "failure" {
		ev.ErrorType = ClassifyError(err)
	}
	return ev
}

func send(apiURL, token string, fromSession bool, ev Event) error {
	payload, err := json.Marshal(map[string]any{"events": []Event{ev}})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.TelemetryTimeout)
	defer cancel()

	if debugEnabled() {
		fmt.Fprintln(os.Stderr, "[telemetry] POST", apiURL+"/v2/cli/telemetry/batch")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		apiURL+"/v2/cli/telemetry/batch", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := constants.NewHTTPClient(constants.TelemetryTimeout)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if debugEnabled() {
		fmt.Fprintln(os.Stderr, "[telemetry] response status:", resp.StatusCode)
	}

	// On 401 and session auth, refresh and retry once
	if resp.StatusCode == http.StatusUnauthorized && fromSession {
		session, loadErr := auth.Load()
		if loadErr != nil {
			return nil
		}
		if session.Refresh(apiURL) != nil {
			return nil
		}
		if auth.Save(session) != nil {
			return nil
		}
		// Retry with new token
		ctx2, cancel2 := context.WithTimeout(context.Background(), constants.TelemetryTimeout)
		defer cancel2()
		req2, _ := http.NewRequestWithContext(ctx2, http.MethodPost,
			apiURL+"/v2/cli/telemetry/batch", bytes.NewReader(payload))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Authorization", "Bearer "+session.AccessToken)
		resp2, err2 := client.Do(req2)
		if err2 != nil {
			return err2
		}
		defer resp2.Body.Close()
		return nil
	}

	return nil
}
