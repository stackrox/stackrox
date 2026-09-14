//go:build test_e2e

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fetchIsReleaseBuild calls /v1/metadata and returns whether Central was built as a release build.
func fetchIsReleaseBuild(t *testing.T) bool {
	t.Helper()
	client := centralgrpc.HTTPClientForCentral(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/v1/metadata", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, resp.Body.Close()) })
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var metadata struct {
		ReleaseBuild bool `json:"release_build"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&metadata))
	return metadata.ReleaseBuild
}

func TestCLIDownload(t *testing.T) {
	t.Run("all platform binaries are downloadable", func(t *testing.T) {
		type binaryCase struct {
			filename string
			magic    []byte
		}

		const (
			// Magic byte sequences that identify each platform's binary format.
			linuxMagic   = "\x7fELF"
			darwinMagic  = "\xcf\xfa\xed\xfe"
			windowsMagic = "MZ"

			// stubPrefix is the known prefix of CI stub files created by
			// .github/workflows/build.yaml in "Create CLI stub files" step
			// for non-native-arch binaries in non-release builds.
			stubPrefix = "This is a placeholder"

			// headerBytes must cover the longest magic sequence (4 bytes) and the stub prefix (21 bytes).
			headerBytes = 32
		)

		testCases := []binaryCase{
			{"roxctl-linux-amd64", []byte(linuxMagic)},
			{"roxctl-linux-arm64", []byte(linuxMagic)},
			{"roxctl-linux-ppc64le", []byte(linuxMagic)},
			{"roxctl-linux-s390x", []byte(linuxMagic)},
			{"roxctl-darwin-amd64", []byte(darwinMagic)},
			{"roxctl-darwin-arm64", []byte(darwinMagic)},
			{"roxctl-windows-amd64.exe", []byte(windowsMagic)},
		}

		releaseBuild := fetchIsReleaseBuild(t)

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
				t.Cleanup(func() { assert.NoError(t, resp.Body.Close()) })

				require.Equal(t, http.StatusOK, resp.StatusCode)

				header := make([]byte, headerBytes)
				_, err = io.ReadFull(resp.Body, header)
				require.NoError(t, err)

				hasMagic := bytes.Equal(header[:len(tc.magic)], tc.magic)

				if releaseBuild || tc.filename == "roxctl-linux-amd64" {
					// Release build contains real binaries for all platforms.
					// In PR CI builds, linux-amd64 is the native runner arch and thus always real.
					assert.True(t, hasMagic, "%s: expected binary magic bytes %x, got: %q",
						tc.filename, tc.magic, header)
				} else if !hasMagic {
					// Non-release builds may serve CI stub files for non-native arches.
					// Anything other than real binary or known stub is unexpected.
					assert.True(t, strings.HasPrefix(string(header), stubPrefix),
						"%s: expected binary magic bytes or CI stub content, got: %q", tc.filename, header)
				}

				_, err = io.Copy(io.Discard, resp.Body)
				require.NoError(t, err)
			})
		}
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
		unauthClient := centralgrpc.UnauthenticatedHTTPClientForCentral(t)
		unauthClient.Timeout = 30 * time.Second

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			"/api/cli/download/roxctl-linux-amd64", nil)
		require.NoError(t, err)

		resp, err := unauthClient.Do(req)
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, resp.Body.Close()) })

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
