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
