// reader.go provides shared io.Reader helpers used by the CLI and MCP packages.
package util

import "io"

// JSONReader implements io.Reader over a pre-encoded JSON byte slice.
// Use it to pass JSON data as stdin to kernel.Execute without a real pipe.
type JSONReader struct {
	data []byte
	pos  int
}

// NewJSONReader returns a JSONReader that reads from data.
func NewJSONReader(data []byte) *JSONReader {
	return &JSONReader{data: data}
}

func (r *JSONReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	if r.pos >= len(r.data) {
		err = io.EOF
	}
	return
}
