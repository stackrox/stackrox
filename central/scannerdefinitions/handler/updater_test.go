package handler

import (
	"archive/zip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mappingURL = "https://definitions.stackrox.io/v4/redhat-repository-mappings/mapping.zip"

	v4VulnURL = "https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulns.json.zst"
)

var (
	nov23 = time.Date(2019, time.November, 23, 0, 0, 0, 0, time.UTC)
)

func assertOnFileExistence(t *testing.T, path string, shouldExist bool) {
	exists, err := fileutils.Exists(path)
	require.NoError(t, err)
	assert.Equal(t, shouldExist, exists)
}

func TestUpdate(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "dump.zip")

	lastUpdatedTime := time.Now().UTC().Truncate(time.Second)
	body := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	responseLengths := [3]int{len(body), len(body), len(body)}
	headContentLengths := [3]int{len(body), len(body), len(body)}
	responseI := 0
	responses := 1
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseI = min(responses-1, responseI)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", headContentLengths[responseI]))
		w.Header().Set("Last-Modified", lastUpdatedTime.Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, body[:responseLengths[responseI]])
		responseI++
	}))

	u := newUpdater(file.New(filePath), &http.Client{Timeout: 30 * time.Second}, server.URL, 1*time.Hour)
	u.RetryDelay = 0

	// Should fetch on first try.
	assert.NotPanics(t, u.update)
	assertOnFileExistence(t, filePath, true)

	// Should not fetch if Last-Modified is not newer than file modified time.
	mustSetModTime(t, filePath, lastUpdatedTime)
	assert.NotPanics(t, u.update)
	assert.Equal(t, lastUpdatedTime, mustGetModTime(t, filePath))
	assertOnFileExistence(t, filePath, true)

	// Should fetch if Last-Modified is newer than file modified time.
	mustSetModTime(t, filePath, nov23)
	assert.NotPanics(t, u.update)
	assert.True(t, mustGetModTime(t, filePath).After(nov23))
	assertOnFileExistence(t, filePath, true)

	// Wrapped doUpdate() should fail, and not change the file, if the downloaded content-length is too short.
	mustSetModTime(t, filePath, nov23)
	responseLengths[0] = len(body) / 2
	responses = 2
	responseI = 0
	require.Error(t, u.doUpdate())
	assert.Equal(t, nov23, mustGetModTime(t, filePath))

	// Should retry, if the downloaded content-length is too short.
	mustSetModTime(t, filePath, nov23)
	responseLengths[0] = len(body) / 2
	responses = 2
	responseI = 0
	assert.NotPanics(t, u.update)
	assert.NotEqual(t, nov23, mustGetModTime(t, filePath))

	// Wrapped doUpdate() should fail, and not change the file, if the downloaded content-length is too long.
	mustSetModTime(t, filePath, nov23)
	responseLengths[0] = len(body)
	headContentLengths[0] = len(body) / 2
	responses = 2
	responseI = 0
	require.Error(t, u.doUpdate())
	assert.Equal(t, nov23, mustGetModTime(t, filePath))

	// Should retry, if the downloaded content-length is too long.
	mustSetModTime(t, filePath, nov23)
	responseLengths[0] = len(body)
	headContentLengths[0] = len(body) / 2
	responses = 2
	responseI = 0
	assert.NotPanics(t, u.update)
	assert.NotEqual(t, nov23, mustGetModTime(t, filePath))
}

func mustGetModTime(t *testing.T, path string) time.Time {
	fi, err := os.Stat(path)
	require.NoError(t, err)
	return fi.ModTime().UTC()
}

func mustSetModTime(t *testing.T, path string, modTime time.Time) {
	require.NoError(t, os.Chtimes(path, time.Now(), modTime))
}

func TestMappingUpdate(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "test.zip")
	u := newUpdater(file.New(filePath), &http.Client{Timeout: 30 * time.Second}, mappingURL, 1*time.Hour)

	// Should fetch first time.
	require.NoError(t, u.doUpdate())
	assertOnFileExistence(t, filePath, true)

	n, err := countFilesInZip(filePath)
	if err != nil {
		t.Fatalf("Failed to count files in zip: %v", err)
	}
	assert.Equal(t, len(v4FileMapping), n)
}

func TestV4VulnUpdate(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "test.json.zst")
	u := newUpdater(file.New(filePath), &http.Client{Timeout: 1 * time.Minute}, v4VulnURL, 1*time.Hour)

	// Should fetch first time.
	require.NoError(t, u.doUpdate())
	assertOnFileExistence(t, filePath, true)
}

// countFilesInZip counts the number of files inside a zip archive.
func countFilesInZip(zipFilePath string) (int, error) {
	r, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Errorf("Error closing zip reader: %v", err)
		}
	}()

	count := 0
	for _, f := range r.File {
		if !f.FileInfo().IsDir() {
			count++
		}
	}

	return count, nil
}
