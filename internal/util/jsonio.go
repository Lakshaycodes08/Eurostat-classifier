package util

import (
	"encoding/json"
	"io"
)

// ReadJSON decodes JSON from the given reader into v.
func ReadJSON(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

// WriteJSON encodes v as JSON to the given writer.
//
// Errors are returned so callers can decide how to surface them
// (e.g. mapped to an internal error exit code).
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

