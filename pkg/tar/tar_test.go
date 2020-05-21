package tar

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testBasePathIn = "tarTestIn"
const testTarHolder = "tar"
const testTarName = "test.tar"
const testBasePathOut = "tarTestOut"

const testFileContents = "test file"

var testStructure = []string{
	"f1",
	"p1/f1",
	"p1/p2/f1",
	"f2",
}

func TestTarUntar(t *testing.T) {
	tmpDirIn, err := ioutil.TempDir("", testBasePathIn)
	assert.NoError(t, err)

	err = createTarDir(t, tmpDirIn)
	assert.NoError(t, err)

	tmpDir, err := ioutil.TempDir("", testTarHolder)
	assert.NoError(t, err)

	f, err := os.OpenFile(tmpDir+testTarName, os.O_CREATE|os.O_RDWR, os.ModePerm)
	assert.NoError(t, err)

	tWriter := tar.NewWriter(f)
	err = FromPath(tmpDirIn, tWriter)
	assert.NoError(t, err)
	err = tWriter.Close()
	assert.NoError(t, err)
	err = f.Close()
	assert.NoError(t, err)

	tmpDirOut, err := ioutil.TempDir("", testBasePathOut)
	assert.NoError(t, err)

	f, err = os.OpenFile(tmpDir+testTarName, os.O_RDONLY, os.ModePerm)
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
		b, err := ioutil.ReadAll(f)
		assert.NoError(t, err)
		assert.Equal(t, string(b), testFileContents)
	}
	return nil
}
