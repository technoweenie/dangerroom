// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// HTTP reverse proxy handler (modified for D)

package dangerroom

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// onExitFlushLoop is a callback set by tests to detect the state of the
// flushLoop() goroutine.
var onExitFlushLoop func()

// Proxy is an HTTP Handler that takes an incoming request and sends it
// to another server, proxying the response back to the client.

// This differs from httputil.ReverseProxy because it creates an HTTP client
// connection to the target, instead of just using the Proxy's Transport.
// This allows it to proxy from HTTP to an HTTPS target.  This is probably
// not ideal in production environments, but the Danger Room is designed for
// testing only.
type Proxy struct {
	Client  *http.Client
	Harness Harness

	// Director must be a function which modifies
	// the request into a new request to be sent
	// using Transport. Its response is then copied
	// back to the original client unmodified.
	Director func(*http.Request)

	// The transport used to perform proxy requests.
	// If nil, http.DefaultTransport is used.
	Transport http.RoundTripper

	// FlushInterval specifies the flush interval
	// to flush to the client while copying the
	// response body.
	// If zero, no periodic flushing is done.
	FlushInterval time.Duration

	// ErrorLog specifies an optional logger for errors
	// that occur when attempting to proxy the request.
	// If nil, logging goes to os.Stderr via the log package's
	// standard logger.
	ErrorLog *log.Logger
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// NewSingleHostProxy returns a new Proxy that rewrites
// URLs to the scheme, host, and base path provided in target. If the
// target's path is "/base" and the incoming request was for "/dir",
// the target request will be for /base/dir.
func NewSingleHostProxy(target *url.URL, prefix string, h Harness, c *http.Client) *Proxy {
	return &Proxy{
		Client:   c,
		Harness:  h,
		Director: NewSingleHostDirector(target, prefix),
	}
}

func NewSingleHostDirector(target *url.URL, prefix string) func(*http.Request) {
	targetQuery := target.RawQuery
	return func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		path := singleJoiningSlash(target.Path, req.URL.Path)
		if prefixLen := len(prefix); prefixLen > 0 {
			req.URL.Path = path[prefixLen+1:]
		} else {
			req.URL.Path = path
		}
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	cli := p.Client
	if cli == nil {
		cli = http.DefaultClient
		p.Client = cli
	}

	h := p.Harness
	if h == nil {
		h = NoopHarness()
		p.Harness = h
	}

	outreq := new(http.Request)
	*outreq = *req     // includes shallow copies of maps, but okay
	p.Director(outreq) // modify request url

	outreq, err := http.NewRequest(req.Method, outreq.URL.String(), req.Body)
	if err != nil {
		log.Printf("http: request error: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	outreq.Header = make(http.Header)
	copyHeader(outreq.Header, req.Header)

	// Remove hop-by-hop headers to the backend.  Especially
	// important is "Connection" because we want a persistent
	// connection, regardless of what the client sent to us.  This
	// is modifying the same underlying map from req (shallow
	// copied above) so we only copy it if necessary.
	for _, h := range hopHeaders {
		if outreq.Header.Get(h) != "" {
			outreq.Header.Del(h)
		}
	}

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		// If we aren't the first proxy retain prior
		// X-Forwarded-For information as a comma+space
		// separated list and fold multiple headers into one.
		if prior, ok := outreq.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		outreq.Header.Set("X-Forwarded-For", clientIP)
	}

	res, err := cli.Do(outreq)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(fmt.Sprintf("http: proxy error: %v\n", err)))
		return
	}
	defer res.Body.Close()

	for _, h := range hopHeaders {
		res.Header.Del(h)
	}

	rwHeader := rw.Header()
	copyHeader(rwHeader, res.Header)

	status := h.WriteHeader(res.StatusCode, rwHeader)

	rw.WriteHeader(status)
	p.copyResponse(rw, res.Body, h)
}

func (p *Proxy) copyResponse(dst io.Writer, src io.Reader, h Harness) {
	if p.FlushInterval != 0 {
		if wf, ok := dst.(writeFlusher); ok {
			mlw := &maxLatencyWriter{
				dst:     wf,
				latency: p.FlushInterval,
				done:    make(chan bool),
			}
			go mlw.flushLoop()
			defer mlw.stop()
			dst = mlw
		}
	}

	if !h.WriteBody(dst, src) {
		io.Copy(dst, src)
	}
}

type writeFlusher interface {
	io.Writer
	http.Flusher
}

type maxLatencyWriter struct {
	dst     writeFlusher
	latency time.Duration

	lk   sync.Mutex // protects Write + Flush
	done chan bool
}

func (m *maxLatencyWriter) Write(p []byte) (int, error) {
	m.lk.Lock()
	defer m.lk.Unlock()
	return m.dst.Write(p)
}

func (m *maxLatencyWriter) flushLoop() {
	t := time.NewTicker(m.latency)
	defer t.Stop()
	for {
		select {
		case <-m.done:
			if onExitFlushLoop != nil {
				onExitFlushLoop()
			}
			return
		case <-t.C:
			m.lk.Lock()
			m.dst.Flush()
			m.lk.Unlock()
		}
	}
}

func (m *maxLatencyWriter) stop() { m.done <- true }
