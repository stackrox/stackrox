package tar

import (
	"archive/tar"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	var errPathEscapes = errors.New("path escapes from parent")
	testCases := map[string]struct {
		path    string
		errorIs error
	}{
		"should block parent directory traversal": {
			path:    "../../etc/evil.txt",
			errorIs: errPathEscapes,
		},
		"should block deep parent traversal": {
			path:    "../../../tmp/evil.txt",
			errorIs: errPathEscapes,
		},
		"should block absolute path traversal": {
			path:    "/../root/evil.txt",
			errorIs: errPathEscapes,
		},
		"should block mixed relative and parent traversal": {
			path:    "subdir/../../etc/evil.txt",
			errorIs: errPathEscapes,
		},
		"should block absolute path to system directory": {
			path:    "/etc/passwd",
			errorIs: errPathEscapes,
		},
		"should block path starting with multiple slashes": {
			path:    "///etc/evil.txt",
			errorIs: errPathEscapes,
		},
		"should block parent traversal with extra slashes": {
			path:    "dir//..//..//etc/evil.txt",
			errorIs: errPathEscapes,
		},
		"should block path with trailing parent reference": {
			path:    "valid/../../..",
			errorIs: errPathEscapes,
		},
		"should allow single dot in path": {
			path:    "./safe.txt",
			errorIs: nil,
		},
		"should allow double dot that stays within bounds": {
			path:    "a/b/../c/file.txt",
			errorIs: nil,
		},
		"should allow relative path without traversal": {
			path:    "deeply/nested/safe/path.txt",
			errorIs: nil,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Create a TAR file with the test entry
			tmpDir := t.TempDir()
			tarPath := filepath.Join(tmpDir, "test.tar")

			f, err := os.Create(tarPath)
			require.NoError(t, err)
			defer utils.IgnoreError(f.Close)

			tWriter := tar.NewWriter(f)

			// Add the test entry
			content := "test content"
			header := &tar.Header{
				Name:     tc.path,
				Mode:     0644,
				Size:     int64(len(content)),
				Typeflag: tar.TypeReg,
			}
			err = tWriter.WriteHeader(header)
			require.NoError(t, err)
			_, err = tWriter.Write([]byte(content))
			require.NoError(t, err)

			err = tWriter.Close()
			require.NoError(t, err)
			err = f.Close()
			require.NoError(t, err)

			// Try to extract to a target directory
			extractDir := t.TempDir()

			f, err = os.Open(tarPath)
			require.NoError(t, err)
			defer utils.IgnoreError(f.Close)

			// Extract - behavior depends on path safety
			gotErr := ToPath(extractDir, f)
			entries, readDirErr := os.ReadDir(extractDir)
			require.NoError(t, readDirErr)

			if tc.errorIs != nil {
				// Assert that extraction failed with expected error
				assert.Error(t, gotErr, "Expected error when extracting path: %s", tc.path)
				assert.ErrorContains(t, gotErr, tc.errorIs.Error(),
					"Error should indicate path traversal was blocked")

				// Defense-in-depth: verify no files escaped the extractDir
				assert.Empty(t, entries, "No files should be extracted when path traversal is detected")
			} else {
				// Assert that extraction succeeded
				assert.NoError(t, gotErr, "Expected successful extraction for safe path: %s", tc.path)

				// Verify file was extracted (path may be cleaned/normalized)
				assert.NotEmpty(t, entries, "File should be extracted for safe path")
			}
		})
	}
}

func TestToPath_HappyPath(t *testing.T) {
	testCases := map[string]struct {
		files    map[string]string
		validate func(t *testing.T, extractDir string)
	}{
		"should extract single file": {
			files: map[string]string{
				"test.txt": "test content",
			},
			validate: func(t *testing.T, extractDir string) {
				content, err := os.ReadFile(filepath.Join(extractDir, "test.txt"))
				assert.NoError(t, err)
				assert.Equal(t, "test content", string(content))
			},
		},
		"should extract nested files": {
			files: map[string]string{
				"dir1/file1.txt":      "content1",
				"dir1/dir2/file2.txt": "content2",
			},
			validate: func(t *testing.T, extractDir string) {
				content1, err := os.ReadFile(filepath.Join(extractDir, "dir1/file1.txt"))
				assert.NoError(t, err)
				assert.Equal(t, "content1", string(content1))

				content2, err := os.ReadFile(filepath.Join(extractDir, "dir1/dir2/file2.txt"))
				assert.NoError(t, err)
				assert.Equal(t, "content2", string(content2))
			},
		},
		"should handle safe path with current dir reference": {
			files: map[string]string{
				"./safe.txt":     "safe content",
				"dir/./file.txt": "nested safe",
			},
			validate: func(t *testing.T, extractDir string) {
				content1, err := os.ReadFile(filepath.Join(extractDir, "safe.txt"))
				assert.NoError(t, err)
				assert.Equal(t, "safe content", string(content1))

				content2, err := os.ReadFile(filepath.Join(extractDir, "dir/file.txt"))
				assert.NoError(t, err)
				assert.Equal(t, "nested safe", string(content2))
			},
		},
		"should handle safe navigation within root": {
			files: map[string]string{
				"dir1/../dir2/file.txt": "content", // navigates but stays within root
			},
			validate: func(t *testing.T, extractDir string) {
				// After cleaning, this should be extracted as "dir2/file.txt"
				content, err := os.ReadFile(filepath.Join(extractDir, "dir2/file.txt"))
				assert.NoError(t, err)
				assert.Equal(t, "content", string(content))
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Create TAR file with specified files
			tmpDir := t.TempDir()
			tarPath := filepath.Join(tmpDir, "test.tar")

			f, err := os.Create(tarPath)
			require.NoError(t, err)
			defer utils.IgnoreError(f.Close)

			tWriter := tar.NewWriter(f)

			for fileName, content := range tc.files {
				header := &tar.Header{
					Name:     fileName,
					Mode:     0644,
					Size:     int64(len(content)),
					Typeflag: tar.TypeReg,
				}
				err = tWriter.WriteHeader(header)
				require.NoError(t, err)
				_, err = tWriter.Write([]byte(content))
				require.NoError(t, err)
			}

			err = tWriter.Close()
			require.NoError(t, err)
			err = f.Close()
			require.NoError(t, err)

			// Extract the TAR
			extractDir := t.TempDir()

			f, err = os.Open(tarPath)
			require.NoError(t, err)
			defer utils.IgnoreError(f.Close)

			err = ToPath(extractDir, f)
			assert.NoError(t, err, "Extraction should succeed for valid paths")

			// Run test-specific validation
			tc.validate(t, extractDir)
		})
	}
}
