package dangerroom

import (
	"io"
	"net/http"
)

var Harnesses map[string]HarnessFunc

type Harness interface {
	WriteHeader(int, http.Header) int
	WriteBody(io.Writer, io.Reader) bool
}

type HarnessFunc func() interface{}

func AddHarness(ctype string, f HarnessFunc) {
	Harnesses[ctype] = f
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

func init() {
	Harnesses = make(map[string]HarnessFunc)
}
