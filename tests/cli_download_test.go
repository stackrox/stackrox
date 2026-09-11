//go:build test_e2e

package tests

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIDownload(t *testing.T) {
	t.Run("all binaries are downloadable", func(t *testing.T) {
		// Magic bytes to identify binary format: ELF (Linux), Mach-O 64-bit LE (Darwin), PE/MZ (Windows).
		var (
			linuxMagic   = []byte{0x7f, 'E', 'L', 'F'}
			darwinMagic  = []byte{0xcf, 0xfa, 0xed, 0xfe}
			windowsMagic = []byte{'M', 'Z'}
		)

		testCases := []struct {
			filename string
			magic    []byte
		}{
			{"roxctl-linux-amd64", linuxMagic},
			{"roxctl-linux-arm64", linuxMagic},
			{"roxctl-linux-ppc64le", linuxMagic},
			{"roxctl-linux-s390x", linuxMagic},
			{"roxctl-darwin-amd64", darwinMagic},
			{"roxctl-darwin-arm64", darwinMagic},
			{"roxctl-windows-amd64.exe", windowsMagic},
		}

		for _, tc := range testCases {
			t.Run(tc.filename, func(t *testing.T) {
				client := centralgrpc.HTTPClientForCentral(t)
				client.Timeout = 2 * time.Minute

				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()

				req, err := http.NewRequestWithContext(ctx, http.MethodGet,
					"/api/cli/download/"+tc.filename, nil)
				require.NoError(t, err)

				resp, err := client.Do(req)
				require.NoError(t, err)
				defer utils.IgnoreError(resp.Body.Close)

				require.Equal(t, http.StatusOK, resp.StatusCode)

				// Read the magic bytes first, then drain the rest to confirm the full stream completes without error.
				firstBytes := make([]byte, len(tc.magic))
				_, err = io.ReadFull(resp.Body, firstBytes)
				require.NoError(t, err)
				assert.Equal(t, tc.magic, firstBytes)
				_, err = io.Copy(io.Discard, resp.Body)
				require.NoError(t, err)
			})
		}
	})

	t.Run("HEAD returns headers with no body", func(t *testing.T) {
		client := centralgrpc.HTTPClientForCentral(t)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodHead,
			"/api/cli/download/roxctl-linux-amd64", nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer utils.IgnoreError(resp.Body.Close)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Greater(t, resp.ContentLength, int64(0))
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Empty(t, body)
	})

	t.Run("POST returns 405 method not allowed", func(t *testing.T) {
		client := centralgrpc.HTTPClientForCentral(t)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			"/api/cli/download/roxctl-linux-amd64", nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer utils.IgnoreError(resp.Body.Close)

		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		endpoint := centralgrpc.RoxAPIEndpoint(t)
		unauthClient := &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //#nosec G402
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			"https://"+endpoint+"/api/cli/download/roxctl-linux-amd64", nil)
		require.NoError(t, err)

		resp, err := unauthClient.Do(req)
		require.NoError(t, err)
		defer utils.IgnoreError(resp.Body.Close)

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
