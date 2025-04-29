package handler

import (
	"archive/zip"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// defURL = "https://definitions.stackrox.io/e799c68a-671f-44db-9682-f24248cd0ffe/diff.zip"

	mappingURL = "https://definitions.stackrox.io/v4/redhat-repository-mappings/mapping.zip"

	v4VulnURL = "https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulns.json.zst"
)

var (
	nov23 = time.Date(2019, time.November, 23, 0, 0, 0, 0, time.Local)
)

func assertOnFileExistence(t *testing.T, path string, shouldExist bool) {
	exists, err := fileutils.Exists(path)
	require.NoError(t, err)
	assert.Equal(t, shouldExist, exists)
}

// TODO(ROX-29099): Make its pass with Scanner V4 enabled
//
// func TestUpdate(t *testing.T) {
// 	filePath := filepath.Join(t.TempDir(), "dump.zip")
// 	u := newUpdater(file.New(filePath), &http.Client{Timeout: 30 * time.Second}, defURL, 1*time.Hour)
// 	// Should fetch first time.
// 	require.NoError(t, u.doUpdate())
// 	assertOnFileExistence(t, filePath, true)

// 	lastUpdatedTime := time.Now()
// 	mustSetModTime(t, filePath, lastUpdatedTime)
// 	// Should not fetch since it can't be updated in a time in the future.
// 	require.NoError(t, u.doUpdate())
// 	assert.Equal(t, lastUpdatedTime.UTC(), mustGetModTime(t, filePath))
// 	assertOnFileExistence(t, filePath, true)

// 	// Should definitely fetch.
// 	mustSetModTime(t, filePath, nov23)
// 	require.NoError(t, u.doUpdate())
// 	assert.True(t, lastUpdatedTime.UTC().After(mustGetModTime(t, filePath)))
// 	assert.True(t, mustGetModTime(t, filePath).After(nov23.UTC()))
// 	assertOnFileExistence(t, filePath, true)
// }

// func mustGetModTime(t *testing.T, path string) time.Time {
// 	fi, err := os.Stat(path)
// 	require.NoError(t, err)
// 	return fi.ModTime().UTC()
// }

// func mustSetModTime(t *testing.T, path string, modTime time.Time) {
// 	require.NoError(t, os.Chtimes(path, time.Now(), modTime))
// }

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
