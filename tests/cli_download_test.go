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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIDownload(t *testing.T) {
	// Magic bytes are only checked for linux-amd64 because PR CI
	// produces a real roxctl binary only for the runner arch (amd64).
	// Other platforms get stub files unless the `ci-build-cli`` label is set on the PR.
	// See .github/workflows/build.yaml "Create CLI stub files" step for details.
	const binaries := []string{
		"roxctl-linux-amd64",
		"roxctl-linux-arm64",
		"roxctl-linux-ppc64le",
		"roxctl-linux-s390x",
		"roxctl-darwin-amd64",
		"roxctl-darwin-arm64",
		"roxctl-windows-amd64.exe",
	}
	// ELF magic bytes identify a valid Linux binary.
	const elfMagic = "\x7fELF"

	for _, filename := range binaries {
		t.Run(filename, func(t *testing.T) {
			client := centralgrpc.HTTPClientForCentral(t)
			client.Timeout = 2 * time.Minute

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet,
				"/api/cli/download/"+filename, nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { assert.NoError(t, resp.Body.Close()) })

			require.Equal(t, http.StatusOK, resp.StatusCode)

			if filename == "roxctl-linux-amd64" {
				// Verify the real binary starts with ELF magic, then drain the rest to confirm
				// the full stream completes without error.
				firstBytes := make([]byte, len(elfMagic))
				_, err = io.ReadFull(resp.Body, firstBytes)
				require.NoError(t, err)
				assert.Equal(t, []byte(elfMagic), firstBytes)
			}
			_, err = io.Copy(io.Discard, resp.Body)
			require.NoError(t, err)
		})
	}

	t.Run("nonexistent binary returns 404", func(t *testing.T) {
		client := centralgrpc.HTTPClientForCentral(t)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			"/api/cli/download/roxctl-nonexistent", nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, resp.Body.Close()) })

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		endpoint := centralgrpc.RoxAPIEndpoint(t)
		// Not using centralgrpc.HTTPClientForCentral(t) because for this test we need the request to be unauthenticated.
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
		t.Cleanup(func() { assert.NoError(t, resp.Body.Close()) })

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
