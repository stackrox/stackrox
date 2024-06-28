package tlscheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_addrValid(t *testing.T) {
	badAddrs := []string{
		// should not contain scheme prefix
		"http://example.com",
		"tcp://example.com",
		"http://exam ple.com:80/abc",
		// should not contain illegal characters
		"exam ple.com:80/abc",
		" example.com",
		"example.com ",
		"exam ple.com/abc",
	}

	for _, addr := range badAddrs {
		t.Run(addr, func(t *testing.T) {
			assert.Error(t, addrValid(addr))
		})
	}

	goodAddrs := []string{
		"example.com:80/abc",
		"127.0.0.1:8080",
		"example.com/repo/path",
		"1::",
		"1::/path",
		"[1::]:80",
		"[1::]:80/path",
		"2001:0db8:0000:0000:0000:ff00:0042:8329",
		"[2001:0db8:0000:0000:0000:ff00:0042:8329]:61273",
		// RFC2732 says we MAY use the format with `[IPv6addr]:port`,
		// but it does not explicitly define the following as invalid.
		// For the sake of simplicity (in using url.Parse), we treat the following as valid.
		"2001:0db8:0000:0000:0000:ff00:0042:8329:61273",
	}

	for _, addr := range goodAddrs {
		t.Run(addr, func(t *testing.T) {
			assert.NoError(t, addrValid(addr))
		})
	}
}

func handler(_ http.ResponseWriter, _ *http.Request) {}

func TestTLS(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(handler))
	defer httpServer.Close()

	tls, err := checkTLSWithRetry(httpServer)
	require.NoError(t, err)
	assert.False(t, tls)

	httpsServer := httptest.NewTLSServer(http.HandlerFunc(handler))
	defer httpsServer.Close()
	tls, err = checkTLSWithRetry(httpsServer)
	require.NoError(t, err)
	assert.False(t, tls)
}

func checkTLSWithRetry(server *httptest.Server) (bool, error) {
	var tls bool
	// Retry the test a few times, sometimes in CI this takes longer than the timeout
	err := retry.WithRetry(
		func() error {
			var err error
			tls, err = CheckTLS(context.Background(), server.Listener.Addr().String())
			return err
		},
		retry.Tries(3),
		retry.BetweenAttempts(func(_ int) {
			time.Sleep(1 * time.Second)
		}),
	)
	return tls, err
}
