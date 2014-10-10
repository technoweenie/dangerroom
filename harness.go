package dangerroom

import (
	"io"
	"net/http"
)

type Harness interface {
	WriteHeader(int, http.Header) int
	WriteBody(io.Writer, io.Reader) bool
}

func NoopHarness() Harness {
	return &noopHarness{}
}

type noopHarness struct{}

func (h *noopHarness) WriteHeader(status int, head http.Header) int {
	return status
}

func (h *noopHarness) WriteBody(writer io.Writer, reader io.Reader) bool {
	return false
}
