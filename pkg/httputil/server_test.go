package httputil

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stallMidBody serves srv on a loopback listener, then opens a raw connection and sends a request
// that announces a body it never finishes sending. It reports how long the server took to hang up,
// and whether it hung up at all before giveUp elapsed.
func stallMidBody(t *testing.T, srv *http.Server, giveUp time.Duration) (time.Duration, bool) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		_ = srv.Serve(listener)
	}()
	t.Cleanup(func() {
		_ = srv.Close()
	})

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close()
	})

	// Complete headers promising 1024 bytes, followed by a single byte of body. The remaining 1023
	// never arrive, leaving the server blocked partway through the request.
	_, err = fmt.Fprintf(conn, "POST /stall HTTP/1.1\r\nHost: %s\r\nContent-Length: 1024\r\n\r\nx",
		listener.Addr().String())
	require.NoError(t, err)

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(giveUp)))

	// Read until the server closes the connection. A server that bounds the read hangs up on its own;
	// one that does not leaves us here until our own deadline fires.
	start := time.Now()
	_, err = io.Copy(io.Discard, conn)
	elapsed := time.Since(start)

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return elapsed, false
	}
	return elapsed, true
}

func TestNewServerBoundsStalledRequestBody(t *testing.T) {
	// The handler returns without touching the body, so the only thing still reading from the client
	// is the drain net/http performs after the handler returns. That is precisely the stretch of the
	// request that ReadHeaderTimeout does not cover.
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("ReadTimeout hangs up on a stalled body", func(t *testing.T) {
		srv := NewServer("", handler)
		require.NotZero(t, srv.ReadTimeout,
			"NewServer must set ReadTimeout; ReadHeaderTimeout alone does not bound the request body")

		// Shrink the production timeouts so the test does not have to wait out DefaultReadTimeout.
		// ReadHeaderTimeout stays comfortably longer than ReadTimeout, so that ReadTimeout is
		// demonstrably the field doing the work.
		srv.ReadHeaderTimeout = 2 * time.Second
		srv.ReadTimeout = 250 * time.Millisecond

		elapsed, hungUp := stallMidBody(t, srv, 10*time.Second)
		assert.True(t, hungUp, "server still held the connection open after %s", elapsed)
	})

	t.Run("ReadHeaderTimeout alone does not", func(t *testing.T) {
		// Regression guard: this is the configuration NewServer replaces. If a future change drops
		// ReadTimeout back out of NewServer, the subtest above starts failing rather than silently
		// degrading to this behaviour.
		srv := &http.Server{
			Handler:           handler,
			ReadHeaderTimeout: DefaultReadHeaderTimeout,
		}

		elapsed, hungUp := stallMidBody(t, srv, 1500*time.Millisecond)
		assert.False(t, hungUp,
			"expected the connection to still be open without ReadTimeout, but the server hung up after %s", elapsed)
	})
}

func TestNewServerDefaults(t *testing.T) {
	srv := NewServer(":8099", nil)

	assert.Equal(t, ":8099", srv.Addr)
	assert.Nil(t, srv.Handler, "a nil handler must be preserved so the server falls back to http.DefaultServeMux")
	assert.Equal(t, DefaultReadHeaderTimeout, srv.ReadHeaderTimeout)
	assert.Equal(t, DefaultReadTimeout, srv.ReadTimeout)
}
