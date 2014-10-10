package dangerroom

import (
	"io"
	"net/http"
)

type LimitingHarness struct {
	ResponseSizeLimit int64 `json:"response_size_limit"`
}

func (h *LimitingHarness) WriteHeader(status int, head http.Header) int {
	return status
}

func (h *LimitingHarness) WriteBody(w io.Writer, r io.Reader) bool {
	if h.ResponseSizeLimit < 1 {
		return false
	}

	io.CopyN(w, r, h.ResponseSizeLimit)

	return true
}

type LimitingHarnessResource struct {
	Target  string           `json:"target"`
	Harness *LimitingHarness `json:"harness"`
}

func init() {
	AddHarness("limiting-harness", func() interface{} { return &LimitingHarnessResource{} })
}
