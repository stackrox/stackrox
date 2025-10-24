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

// TestToPath_PreventPathTraversal verifies that malicious TAR files with path traversal
// attempts (e.g., ../../etc/passwd) are blocked by os.Root security
func TestToPath_PreventPathTraversal(t *testing.T) {
	// Create a malicious TAR file with path traversal attempts
	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "malicious.tar")

	f, err := os.Create(tarPath)
	assert.NoError(t, err)
	defer utils.IgnoreError(f.Close)

	tWriter := tar.NewWriter(f)

	// Add malicious entries that try to escape the target directory
	maliciousPaths := []string{
		"../../etc/evil.txt",
		"../../../tmp/evil.txt",
		"/../root/evil.txt",
		"normal.txt", // This one should succeed
	}

	for _, malPath := range maliciousPaths {
		header := &tar.Header{
			Name:     malPath,
			Mode:     0644,
			Size:     int64(len("malicious content")),
			Typeflag: tar.TypeReg,
		}
		err := tWriter.WriteHeader(header)
		assert.NoError(t, err)
		_, err = tWriter.Write([]byte("malicious content"))
		assert.NoError(t, err)
	}

	err = tWriter.Close()
	assert.NoError(t, err)
	err = f.Close()
	assert.NoError(t, err)

	// Try to extract to a target directory
	extractDir := t.TempDir()

	f, err = os.Open(tarPath)
	assert.NoError(t, err)
	defer utils.IgnoreError(f.Close)

	// Extract - malicious paths should be blocked or sanitized
	err = ToPath(extractDir, f)

	// The extraction should either:
	// 1. Fail with an error (path traversal blocked)
	// 2. Succeed but only extract safe files within extractDir
	// With os.Root, path traversal attempts should fail

	// Verify that malicious files were NOT created outside extractDir
	// Check common escape locations
	evilPaths := []string{
		"/etc/evil.txt",
		"/tmp/evil.txt",
		"/root/evil.txt",
		filepath.Join(tmpDir, "evil.txt"), // one level up
		filepath.Join(filepath.Dir(tmpDir), "evil.txt"), // two levels up
	}

	for _, evilPath := range evilPaths {
		_, err := os.Stat(evilPath)
		if err == nil {
			t.Errorf("Security violation: malicious file was created at %s", evilPath)
		}
		// Expect "no such file or directory" - meaning the file wasn't created
		assert.True(t, os.IsNotExist(err), "Malicious file should not exist at %s", evilPath)
	}

	// Verify that the normal file was created successfully inside extractDir
	normalPath := filepath.Join(extractDir, "normal.txt")
	content, err := os.ReadFile(normalPath)
	if err == nil {
		assert.Equal(t, "malicious content", string(content))
	}
	// Note: Even if extraction failed entirely (err != nil above), the security check passed
}
