package handler

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"net/http"
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
	defURL = "https://definitions.stackrox.io/e799c68a-671f-44db-9682-f24248cd0ffe/diff.zip"

	mappingURL = "https://storage.googleapis.com/scanner-v4-test/redhat-repository-mappings/mapping.zip"

	cvssURL = "https://storage.googleapis.com/scanner-v4-test/nvd-bundle/nvd-data.tar.gz"
)

var (
	nov23 = time.Date(2019, time.November, 23, 0, 0, 0, 0, time.Local)
)

func assertOnFileExistence(t *testing.T, path string, shouldExist bool) {
	exists, err := fileutils.Exists(path)
	require.NoError(t, err)
	assert.Equal(t, shouldExist, exists)
}

func TestUpdate(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "dump.zip")
	u := newUpdater(file.New(filePath), &http.Client{Timeout: 30 * time.Second}, defURL, 1*time.Hour)
	// Should fetch first time.
	require.NoError(t, u.doUpdate())
	assertOnFileExistence(t, filePath, true)

	lastUpdatedTime := time.Now().Add(time.Hour)
	mustSetModTime(t, filePath, lastUpdatedTime)
	// Should not fetch since it can't be updated in a time in the future.
	require.NoError(t, u.doUpdate())
	assert.Equal(t, lastUpdatedTime.UTC(), mustGetModTime(t, filePath))
	assertOnFileExistence(t, filePath, true)

	// Should definitely fetch.
	mustSetModTime(t, filePath, nov23)
	require.NoError(t, u.doUpdate())
	assert.True(t, lastUpdatedTime.UTC().After(mustGetModTime(t, filePath)))
	assert.True(t, mustGetModTime(t, filePath).After(nov23.UTC()))
	assertOnFileExistence(t, filePath, true)
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
	u := newV4Updater(file.New(filePath), &http.Client{Timeout: 30 * time.Second}, mappingURL, 1*time.Hour)

	// Should fetch first time.
	require.NoError(t, u.doUpdate())
	assertOnFileExistence(t, filePath, true)

	n, err := countFilesInZip(filePath)
	if err != nil {
		t.Fatalf("Failed to count files in zip: %v", err)
	}
	assert.Equal(t, n, 2)
}

func TestCvssUpdate(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "test.tar.gz")
	u := newV4Updater(file.New(filePath), &http.Client{Timeout: 30 * time.Second}, cvssURL, 1*time.Hour)

	// Should fetch first time.
	require.NoError(t, u.doUpdate())
	assertOnFileExistence(t, filePath, true)

	n, err := countFilesInTarGz(filePath)
	if err != nil {
		t.Fatalf("Failed to count files in zip: %v", err)
	}
	assert.Greater(t, n, 21, "Number of files should be greater than 21")
}

// countFilesInZip counts the number of files inside a zip archive.
func countFilesInZip(zipFilePath string) (int, error) { /**/
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

func countFilesInTarGz(tarGzFilePath string) (int, error) {
	file, err := os.Open(tarGzFilePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return 0, err
	}
	defer gzr.Close()

	tarr := tar.NewReader(gzr)

	count := 0
	for {
		header, err := tarr.Next()
		if err != nil {
			if err == io.EOF {
				break // End of archive
			}
			return 0, err
		}
		if header.Typeflag != tar.TypeDir {
			count++
		}
	}

	return count, nil
}
