package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	defURL = "https://definitions.stackrox.io/e799c68a-671f-44db-9682-f24248cd0ffe/diff.zip"
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
	lastUpdatedTime := time.Now().Add(time.Minute)
	filePath := filepath.Join(t.TempDir(), "dump.zip")
	mustCreate(t, filePath, lastUpdatedTime)
	u := newUpdater(filePath, &http.Client{Timeout: 30 * time.Second}, defURL, 1*time.Hour)
	// Should not fetch since it can't be updated in a time in the future.
	require.NoError(t, u.doUpdate())
	assert.Equal(t, lastUpdatedTime.UTC(), mustStat(t, filePath).ModTime().UTC())

	// Should definitely fetch.
	mustSetModifiedTime(t, filePath, nov23)
	require.NoError(t, u.doUpdate())
	assert.True(t, mustStat(t, filePath).ModTime().After(nov23.UTC()))
	assertOnFileExistence(t, filePath, true)
}

func mustCreate(t *testing.T, path string, modTime time.Time) {
	_, err := os.Create(path)
	require.NoError(t, err)
	mustSetModifiedTime(t, path, modTime)
}

func mustStat(t *testing.T, path string) os.FileInfo {
	fi, err := os.Stat(path)
	require.NoError(t, err)
	return fi
}
