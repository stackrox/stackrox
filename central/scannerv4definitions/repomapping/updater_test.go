package repomapping

import (
	"archive/zip"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

const (
	defURL = "https://storage.googleapis.com/scanner-v4-test/redhat-repository-mappings/"
)

func assertOnFileExistence(t *testing.T, path string, shouldExist bool) {
	exists, err := fileutils.Exists(path)
	require.NoError(t, err)
	assert.Equal(t, shouldExist, exists)
}

func TestUpdate(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "repo.zip")
	u := NewUpdater(file.New(filePath), &http.Client{Timeout: 30 * time.Second}, defURL, 1*time.Hour)

	// Should fetch first time.
	require.NoError(t, u.doUpdate())
	assertOnFileExistence(t, filePath, true)

	n, err := countFilesInZip(filePath)
	if err != nil {
		t.Fatalf("Failed to count files in zip: %v", err)
	}
	assert.Equal(t, n, 2)
}

// countFilesInZip counts the number of files inside a zip archive.
func countFilesInZip(zipFilePath string) (int, error) {
	r, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return 0, err
	}
	defer r.Close()

	count := 0
	for _, f := range r.File {
		if !f.FileInfo().IsDir() {
			count++
		}
	}

	return count, nil
}
