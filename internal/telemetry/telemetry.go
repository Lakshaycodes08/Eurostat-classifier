// telemetry.go sends CLI execution events to the backend in a fire-and-forget goroutine.
// Failures are silently discarded — telemetry must never break the primary command.
package telemetry

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Event represents a single CLI execution event sent to POST /v2/cli/telemetry/batch.
type Event struct {
	EventID     string `json:"event_id"`
	Command     string `json:"command"`               // "check" | "inspect" | "upgrade" | "fetch"
	ProjectUUID string `json:"project_uuid,omitempty"`
	LibraryName string `json:"library_name,omitempty"`
	Outcome     string `json:"outcome"`               // "success" | "failure" | "cancelled"
	CLIVersion  string `json:"cli_version"`
}

// newEventID generates a random UUID v4 string.
func newEventID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// Send fires off a telemetry event asynchronously. It returns immediately; any
// network or serialisation error is silently dropped.
func Send(apiURL, token string, ev Event) {
	if ev.EventID == "" {
		ev.EventID = newEventID()
	}
	go func() {
		if err := send(apiURL, token, ev); err != nil {
			// Intentionally silent — telemetry failures must not surface to users.
			_ = fmt.Sprintf("telemetry: %v", err)
		}
	}()
}

func send(apiURL, token string, ev Event) error {
	payload, err := json.Marshal(map[string]any{"events": []Event{ev}})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		apiURL+"/v2/cli/telemetry/batch", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
