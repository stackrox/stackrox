package vsockserver

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/virtualmachine/cpemapping"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dummyMappingURL is never dialed in these tests: NewURLUpdater's
// constructor only builds the filedownloader.Downloader, and the actual
// fetch is exercised indirectly via onDownloadComplete.
const dummyMappingURL = "http://127.0.0.1:0/mapping.json"

func TestNewURLUpdater_BootstrapFromCache(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")
	writeFile(t, cachePath, validMappingJSON)
	counter := &onChangeCounter{}

	u := NewURLUpdater(dummyMappingURL, cachePath, counter.fn)

	require.True(t, u.Ready())
	assert.Equal(t, cpemapping.HashMapping([]byte(validMappingJSON)), u.Hash())
	b, err := u.Bytes()
	require.NoError(t, err)
	assert.Equal(t, validMappingJSON, string(b))
	path, err := u.Path()
	require.NoError(t, err)
	assert.Equal(t, cachePath, path)
	assert.Equal(t, 1, counter.count, "onChange should fire once for a valid bootstrap")
}

func TestNewURLUpdater_NotReadyWithoutCache(t *testing.T) {
	dir := t.TempDir()
	counter := &onChangeCounter{}

	u := NewURLUpdater(dummyMappingURL, filepath.Join(dir, "cache.json"), counter.fn)

	assert.False(t, u.Ready())
	assert.Empty(t, u.Hash())
	assert.Equal(t, 0, counter.count)
	_, err := u.Bytes()
	assert.Error(t, err)
	_, err = u.Path()
	assert.Error(t, err)
}

func TestNewURLUpdater_InvalidCacheStaysNotReady_NoBundledFallback(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")
	writeFile(t, cachePath, invalidMappingJSON)
	counter := &onChangeCounter{}

	// The URL updater has no bundled-file parameter and no secondary
	// source at all: an invalid cachePath must leave it not Ready, not
	// fall back to anything else.
	u := NewURLUpdater(dummyMappingURL, cachePath, counter.fn)

	assert.False(t, u.Ready())
	assert.Equal(t, 0, counter.count)
}

func TestURLUpdater_UpdatePath_IsURL(t *testing.T) {
	u := NewURLUpdater(dummyMappingURL, filepath.Join(t.TempDir(), "cache.json"), func() {})
	assert.Equal(t, pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_URL, u.UpdatePath())
}

func TestURLUpdater_FailedDownload_StaysNotReady(t *testing.T) {
	counter := &onChangeCounter{}
	u := NewURLUpdater(dummyMappingURL, filepath.Join(t.TempDir(), "cache.json"), counter.fn)

	u.onDownloadComplete(errors.New("connection refused"), 0)

	assert.False(t, u.Ready())
	assert.Equal(t, 0, counter.count)
}

func TestURLUpdater_FailedDownload_KeepsLastGood(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")
	writeFile(t, cachePath, validMappingJSON)
	counter := &onChangeCounter{}
	u := NewURLUpdater(dummyMappingURL, cachePath, counter.fn)
	require.True(t, u.Ready())
	require.Equal(t, 1, counter.count)

	u.onDownloadComplete(errors.New("connection refused"), 0)

	assert.True(t, u.Ready())
	assert.Equal(t, cpemapping.HashMapping([]byte(validMappingJSON)), u.Hash())
	assert.Equal(t, 1, counter.count, "onChange must not re-fire when a failed refresh keeps the same content")
}

func TestURLUpdater_SuccessfulDownload_AppliesNewContent(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")
	counter := &onChangeCounter{}
	u := NewURLUpdater(dummyMappingURL, cachePath, counter.fn)
	require.False(t, u.Ready())

	// Simulate what filedownloader's atomic write already did before
	// invoking onComplete: the new content is on disk at stagingPath.
	writeFile(t, u.stagingPath, validMappingJSON)
	u.onDownloadComplete(nil, 0)

	require.True(t, u.Ready())
	assert.Equal(t, cpemapping.HashMapping([]byte(validMappingJSON)), u.Hash())
	b, err := u.Bytes()
	require.NoError(t, err)
	assert.Equal(t, validMappingJSON, string(b))
	assert.Equal(t, 1, counter.count)

	onDisk, err := os.ReadFile(cachePath)
	require.NoError(t, err)
	assert.Equal(t, validMappingJSON, string(onDisk), "a validated download must be promoted to cachePath")
}

func TestURLUpdater_SuccessfulDownload_InvalidContentKeepsLastGood(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")
	writeFile(t, cachePath, validMappingJSON)
	counter := &onChangeCounter{}
	u := NewURLUpdater(dummyMappingURL, cachePath, counter.fn)
	require.Equal(t, 1, counter.count)

	writeFile(t, u.stagingPath, invalidMappingJSON)
	u.onDownloadComplete(nil, 0)

	assert.True(t, u.Ready())
	assert.Equal(t, cpemapping.HashMapping([]byte(validMappingJSON)), u.Hash())
	assert.Equal(t, 1, counter.count, "onChange must not fire when the refreshed content fails validation")
}

// TestURLUpdater_SuccessfulDownload_InvalidContentDoesNotCorruptCache covers
// what TestURLUpdater_SuccessfulDownload_InvalidContentKeepsLastGood only
// checks in memory: an invalid refresh must leave the persisted cachePath
// itself untouched, since a restart bootstraps from that file, not memory.
func TestURLUpdater_SuccessfulDownload_InvalidContentDoesNotCorruptCache(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")
	writeFile(t, cachePath, validMappingJSON)
	u := NewURLUpdater(dummyMappingURL, cachePath, func() {})

	writeFile(t, u.stagingPath, invalidMappingJSON)
	u.onDownloadComplete(nil, 0)

	onDisk, err := os.ReadFile(cachePath)
	require.NoError(t, err)
	assert.Equal(t, validMappingJSON, string(onDisk), "an invalid refresh must not overwrite the persisted last-good cache")

	restarted := NewURLUpdater(dummyMappingURL, cachePath, func() {})
	assert.True(t, restarted.Ready(), "a restart must still bootstrap from the untouched cache")
}
