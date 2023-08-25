package tar

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
)

const (
	testTarName      = "test_path.tar"
	testFileContents = "test file"
)

var testStructure = []string{
	"f1",
	"p1/f1",
	"p1/p2/f1",
	"f2",
}

func TestFromRelativePathTarUntar(t *testing.T) {
	doTestTarUntar(t, func(dirIn string, tWriter *tar.Writer) {
		testMap := make(map[string]string)
		for _, p := range testStructure {
			testMap[p] = filepath.Join(dirIn, p)
		}
		err := FromPathMap(testMap, tWriter)
		assert.NoError(t, err)
	})
}

func TestFromPathTarUntar(t *testing.T) {
	doTestTarUntar(t, func(dirIn string, tWriter *tar.Writer) {
		err := FromPath(dirIn, tWriter)
		assert.NoError(t, err)
	})
}

func doTestTarUntar(t *testing.T, tarFunc func(string, *tar.Writer)) {
	tmpDirIn := t.TempDir()

	err := createTarDir(t, tmpDirIn)
	assert.NoError(t, err)

	tmpDir := t.TempDir()

	f, err := os.OpenFile(filepath.Join(tmpDir, testTarName), os.O_CREATE|os.O_RDWR, os.ModePerm)
	defer utils.IgnoreError(f.Close)
	assert.NoError(t, err)

	tWriter := tar.NewWriter(f)
	defer utils.IgnoreError(tWriter.Close)
	tarFunc(tmpDirIn, tWriter)
	err = tWriter.Close()
	assert.NoError(t, err)
	err = f.Close()
	assert.NoError(t, err)

	tmpDirOut := t.TempDir()

	f, err = os.OpenFile(filepath.Join(tmpDir, testTarName), os.O_RDONLY, os.ModePerm)
	defer utils.IgnoreError(f.Close)
	assert.NoError(t, err)
	err = ToPath(tmpDirOut, f)
	assert.NoError(t, err)
	err = f.Close()
	assert.NoError(t, err)

	err = checkUntarDir(t, tmpDirOut)
	assert.NoError(t, err)
}

func createTarDir(t *testing.T, path string) error {
	for _, fPath := range testStructure {
		fullPath := filepath.Join(path, fPath)
		// Create the path for the file
		dir := filepath.Dir(fullPath)
		if dir != "" {
			if _, err := os.Stat(dir); err != nil {
				err := os.MkdirAll(dir, 0755)
				assert.NoError(t, err)
			}
		}

		// Create a file
		f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_RDWR, os.ModePerm)
		assert.NoError(t, err)
		_, err = io.Copy(f, strings.NewReader(testFileContents))
		assert.NoError(t, err)
		err = f.Close()
		assert.NoError(t, err)
	}
	return nil
}

func checkUntarDir(t *testing.T, path string) error {
	for _, fPath := range testStructure {
		fullPath := filepath.Join(path, fPath)
		// Create the path for the file
		dir := filepath.Dir(fullPath)
		if dir != "" {
			_, err := os.Stat(dir)
			assert.NoError(t, err)
		}

		// Create a file
		f, err := os.OpenFile(fullPath, os.O_RDONLY, os.ModePerm)
		assert.NoError(t, err)
		b, err := io.ReadAll(f)
		assert.NoError(t, err)
		assert.Equal(t, string(b), testFileContents)
	}
	return nil
}
