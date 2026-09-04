package httputil

import (
	"net/http"
	"time"
)

const (
	// DefaultReadHeaderTimeout bounds how long a server waits for a client to send the request headers.
	DefaultReadHeaderTimeout = 5 * time.Second

	// DefaultReadTimeout bounds how long a server waits for a client to send a complete request,
	// headers and body. ReadHeaderTimeout alone does not cover the body: net/http arms the read
	// deadline only until the headers are parsed, and then resets it to ReadTimeout, so leaving
	// ReadTimeout unset clears the deadline entirely for the rest of the request. That includes the
	// drain net/http performs after the handler returns to make the connection reusable, so a client
	// that announces a Content-Length and then stalls can pin a connection open indefinitely even
	// against a handler that never touches the body.
	DefaultReadTimeout = 60 * time.Second
)

// NewServer returns an *http.Server for addr whose read timeouts bound how long a stalled or
// malicious client can hold a connection open. A nil handler means http.DefaultServeMux, matching
// http.Server itself.
//
// Only the read side is bounded. WriteTimeout is deliberately left unset because it caps the total
// duration of legitimate slow responses, such as large file downloads or pprof profiles. IdleTimeout
// is left unset because net/http falls back to ReadTimeout when it is zero, which is what we want.
func NewServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
		ReadTimeout:       DefaultReadTimeout,
	}
}
