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

func Test_sanitizeURL(t *testing.T) {
	const strInvalidChar = `invalid character " " in host name`
	tests := map[string]struct {
		URL              string
		wantURL          string
		wantErrToContain string
	}{
		"Valid URL with scheme, host and port": {
			URL:              "http://example.com:80/abc",
			wantURL:          "http://example.com:80/abc",
			wantErrToContain: "",
		},
		"Valid URL with host and port": {
			URL:              "example.com:80/abc",
			wantURL:          "tcp://example.com:80/abc",
			wantErrToContain: "",
		},
		"Valid URL with host": {
			URL:              "example.com",
			wantURL:          "tcp://example.com",
			wantErrToContain: "",
		},
		"Valid URL with IP as host": {
			URL:              "192.168.178.1",
			wantURL:          "tcp://192.168.178.1",
			wantErrToContain: "",
		},
		"URL with scheme, port and space in host": {
			URL:              "http://exam ple.com:80/abc",
			wantURL:          "",
			wantErrToContain: strInvalidChar,
		},
		"URL with port and space in host": {
			URL:              "exam ple.com:80/abc",
			wantErrToContain: "first path segment in URL cannot contain colon",
		},
		"URL with scheme, and space in host": {
			URL:              "tcp://exam ple.com/abc",
			wantErrToContain: strInvalidChar,
		},
		"URL with leading space in host": {
			URL:              " example.com",
			wantErrToContain: strInvalidChar,
		},
		"URL with trailing space in host": {
			URL:              "example.com ",
			wantErrToContain: strInvalidChar,
		},
		"URL with space in host": {
			URL:              "exam ple.com/abc",
			wantErrToContain: strInvalidChar,
		},
	}
	for tname, tt := range tests {
		t.Run(tname, func(t *testing.T) {
			gotAddr, gotErr := validateWithScheme(tt.URL)
			if tt.wantErrToContain == "" {
				assert.NoError(t, gotErr)
				assert.Equal(t, tt.wantURL, gotAddr)
			} else {
				assert.ErrorContains(t, gotErr, tt.wantErrToContain)
			}
		})
	}
}
