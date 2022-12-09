package profiling

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_fifoDir(t *testing.T) {
	dir := t.TempDir()
	maxFileCount := 3
	fs := fifoDir{dirPath: dir, maxFileCount: maxFileCount}

	var filesToBeDeleted []string
	var filesToBeKept []string
	// This should be bigger than maxFileCount to test FIFO deletion is done properly
	numFilesToCreate := 10
	for i := 0; i < numFilesToCreate; i++ {
		fileName := fmt.Sprintf("%d.test.dump", i)
		_, err := fs.Create(fileName)
		require.NoError(t, err, "creating file: %s", fileName)
		if i < numFilesToCreate-3 {
			filesToBeDeleted = append(filesToBeDeleted, fileName)
		} else {
			filesToBeKept = append(filesToBeKept, fileName)
		}
		time.Sleep(time.Millisecond * 100)
	}

	actualFilesEntries, err := os.ReadDir(dir)
	require.NoErrorf(t, err, "reading directory %s", dir)
	actualFiles, err := dirEntriesToFileInfo(actualFilesEntries)
	require.NoErrorf(t, err, "convert get []fs.FileInfo from []os.DirEntry")
	require.Equalf(t, len(actualFiles), maxFileCount, "file count in given directory should be equal to maxFileCount")

	actualFileNames := make([]string, len(actualFiles))
	for i := range actualFiles {
		actualFileNames[i] = actualFiles[i].Name()
	}

	for _, f := range filesToBeDeleted {
		assert.NotContains(t, actualFileNames, f)
	}

	for _, f := range filesToBeKept {
		assert.Contains(t, actualFileNames, f)
	}
}
