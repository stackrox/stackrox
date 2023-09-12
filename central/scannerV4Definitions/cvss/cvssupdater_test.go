package cvss

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

const (
	defURL = "https://storage.googleapis.com/scanner-v4-test/nvddata/"
)

func assertOnFileExistence(t *testing.T, path string, shouldExist bool) {
	exists, err := fileutils.Exists(path)
	require.NoError(t, err)
	assert.Equal(t, shouldExist, exists)
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	filePath := filepath.Join(t.TempDir(), "cvss.zip")
	u, err := NewUpdaterWithEnricher(file.New(filePath), &http.Client{Timeout: 30 * time.Second}, defURL, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create new updater with enricher: %v", err)
	}
	// Should fetch first time.
	require.NoError(t, u.doUpdate(ctx))
	assertOnFileExistence(t, filePath, true)
}

func TestUpdateMemory(t *testing.T) {
	ctx := context.Background()
	filePath := filepath.Join(t.TempDir(), "cvss.zip")
	u, err := NewUpdaterWithEnricher(file.New(filePath), &http.Client{Timeout: 30 * time.Second}, defURL, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create new updater with enricher: %v", err)
	}

	// Measure memory before
	var memStart, memEnd runtime.MemStats
	runtime.ReadMemStats(&memStart)

	// Call the function
	require.NoError(t, u.doUpdate(ctx))

	// Measure memory after
	runtime.ReadMemStats(&memEnd)

	// Calculate and print the difference
	allocDiff := memEnd.Alloc - memStart.Alloc
	totalAllocDiff := memEnd.TotalAlloc - memStart.TotalAlloc

	fmt.Printf("Memory Alloc (difference): %v bytes\n", allocDiff)
	fmt.Printf("Memory TotalAlloc (difference): %v bytes\n", totalAllocDiff)

	assertOnFileExistence(t, filePath, true)
}
