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
	// Only linux-amd64 is tested here because PR CI builds only produce a real roxctl binary for
	// the native build runner architecture (amd64). Other platforms get stub files unless the
	// `ci-build-cli`` label is set on the PR.
	// See .github/workflows/build.yaml
	t.Run("linux-amd64 binary is downloadable", func(t *testing.T) {
		// ELF magic bytes identify a valid Linux binary.
		const elfMagic = "\x7fELF"

		client := centralgrpc.HTTPClientForCentral(t)
		client.Timeout = 2 * time.Minute

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			"/api/cli/download/roxctl-linux-amd64", nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, resp.Body.Close()) })

		require.Equal(t, http.StatusOK, resp.StatusCode)

		// Read the magic bytes first, then drain the rest to confirm the full stream completes without error.
		firstBytes := make([]byte, len(elfMagic))
		_, err = io.ReadFull(resp.Body, firstBytes)
		require.NoError(t, err)
		assert.Equal(t, []byte(elfMagic), firstBytes)
		_, err = io.Copy(io.Discard, resp.Body)
		require.NoError(t, err)
	})

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
