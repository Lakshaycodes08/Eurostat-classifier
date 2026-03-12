package telemetry

import (
	"errors"
	"testing"
)

func TestBuildEvent_SetsEventID(t *testing.T) {
	ev1 := buildEvent("exec", "tool.one", nil, nil)
	if ev1.EventID == "" {
		t.Fatalf("expected non-empty EventID for success event")
	}

	ev2 := buildEvent("exec", "tool.two", errors.New("boom"), nil)
	if ev2.EventID == "" {
		t.Fatalf("expected non-empty EventID for failure event")
	}

	if ev1.EventID == ev2.EventID {
		t.Fatalf("expected different EventIDs for different events, got %q", ev1.EventID)
	}
}

func TestSendEvent_UsesBuildEventEventID(t *testing.T) {
	// We can't easily intercept the HTTP call here, but we can at least assert that
	// buildEvent (used by SendEvent) produces a populated EventID, which is what
	// the backend will see once marshalled.
	ev := buildEvent("check", "tool.lib", errors.New("err"), &EventOpts{})
	if ev.EventID == "" {
		t.Fatalf("expected non-empty EventID from buildEvent for SendEvent path")
	}
}

