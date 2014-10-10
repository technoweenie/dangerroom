package dangerroom

import (
	"io"
	"net/http"
)

type LimitingHarness struct {
	ResponseSizeLimit int64
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
