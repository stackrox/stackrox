package profiling

import (
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

	// This should be more files than maxFileCount to test FIFO deletion is done properly
	filesToCreate := []string{"1.test.dump", "2.test.dump", "3.test.dump", "4.test.dump", "5.test.dump"}
	filesToBeDeleted := filesToCreate[:2]
	filesToBeKept := filesToCreate[2:]

	for _, fileName := range filesToCreate {
		_, err := fs.Create(fileName)
		time.Sleep(time.Millisecond * 100) // to prevent flakes due to inconsistent ordering in case FS time resolution is low
		require.NoError(t, err, "creating file: %s", fileName)
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
