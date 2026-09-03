package vuln

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGitHubCIMinimalBundleAccessible verifies that the ci-minimal bundle
// hosted on GitHub is accessible via HTTP and can be fetched by the updater.
// This test ensures the URL used in CI configurations remains valid.
//
// This test requires network access and will be skipped in restricted environments.
func TestGitHubCIMinimalBundleAccessible(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network-dependent test in short mode")
	}

	const githubBundleURL = "https://raw.githubusercontent.com/stackrox/stackrox/604dc32bb0/scanner/image/scanner/bundles/ci-minimal/vulnerabilities.zip"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubBundleURL, nil)
	require.NoError(t, err, "failed to create HTTP request")

	req.Header.Set("If-Modified-Since", time.Time{}.Format(http.TimeFormat))
	req.Header.Set("X-Scanner-V4-Accept", "application/vnd.stackrox.scanner-v4.multi-bundle+zip")

	resp, err := client.Do(req)
	require.NoError(t, err, "failed to fetch bundle from GitHub")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 OK status")
	assert.Equal(t, "application/zip", resp.Header.Get("Content-Type"), "expected application/zip content type")

	// Verify the response contains data (bundle should be ~18KB)
	buffer := make([]byte, 1024)
	n, err := resp.Body.Read(buffer)
	require.NoError(t, err, "failed to read response body")
	assert.Greater(t, n, 0, "response body should not be empty")

	// Check for ZIP file magic bytes (PK\x03\x04)
	assert.Equal(t, byte('P'), buffer[0], "expected ZIP magic byte 'P'")
	assert.Equal(t, byte('K'), buffer[1], "expected ZIP magic byte 'K'")
	assert.Equal(t, byte(0x03), buffer[2], "expected ZIP magic byte 0x03")
	assert.Equal(t, byte(0x04), buffer[3], "expected ZIP magic byte 0x04")
}
