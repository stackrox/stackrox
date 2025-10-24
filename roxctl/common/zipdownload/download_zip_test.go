package zipdownload

import (
	"archive/zip"
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testLogger struct {
	t *testing.T
}

func (l *testLogger) InfofLn(format string, args ...interface{}) { l.t.Logf(format, args...) }
func (l *testLogger) WarnfLn(format string, args ...interface{}) { l.t.Logf("WARN: "+format, args...) }
func (l *testLogger) ErrfLn(format string, args ...interface{})  { l.t.Logf("ERROR: "+format, args...) }
func (l *testLogger) ErrorfLn(format string, args ...interface{}) {
	l.t.Logf("ERROR: "+format, args...)
}
func (l *testLogger) PrintfLn(format string, args ...interface{}) { l.t.Logf(format, args...) }

// TestExtractZipToFolder_HappyPaths verifies ZIP extraction works correctly for various valid scenarios
func TestExtractZipToFolder_HappyPaths(t *testing.T) {
	testCases := map[string]struct {
		files    map[string]string
		validate func(t *testing.T, extractDir string, files map[string]string)
	}{
		"should extract multiple files with nested directories": {
			files: map[string]string{
				"file1.txt":           "content1",
				"dir1/file2.txt":      "content2",
				"dir1/dir2/file3.txt": "content3",
			},
			validate: func(t *testing.T, extractDir string, files map[string]string) {
				for name, expectedContent := range files {
					filePath := filepath.Join(extractDir, name)
					content, err := os.ReadFile(filePath)
					assert.NoError(t, err, "File %s should exist", name)
					assert.Equal(t, expectedContent, string(content), "Content mismatch for %s", name)
				}
			},
		},
		"should create deeply nested directory structure": {
			files: map[string]string{
				"level1/level2/level3/level4/file.txt": "deep content",
			},
			validate: func(t *testing.T, extractDir string, files map[string]string) {
				filePath := filepath.Join(extractDir, "level1/level2/level3/level4/file.txt")
				content, err := os.ReadFile(filePath)
				assert.NoError(t, err)
				assert.Equal(t, "deep content", string(content))
			},
		},
		"should handle empty ZIP without error": {
			files: map[string]string{},
			validate: func(t *testing.T, extractDir string, files map[string]string) {
				entries, err := os.ReadDir(extractDir)
				assert.NoError(t, err)
				assert.Empty(t, entries)
			},
		},
		"should handle explicit directory entries": {
			files: map[string]string{
				"emptydir/":    "", // Explicit directory entry
				"dir/file.txt": "content",
				"dir/subdir/":  "", // Explicit subdirectory entry
			},
			validate: func(t *testing.T, extractDir string, files map[string]string) {
				// Verify directories were created
				dirPath := filepath.Join(extractDir, "emptydir")
				info, err := os.Stat(dirPath)
				assert.NoError(t, err, "emptydir should exist")
				assert.True(t, info.IsDir(), "emptydir should be a directory")

				subdirPath := filepath.Join(extractDir, "dir/subdir")
				info, err = os.Stat(subdirPath)
				assert.NoError(t, err, "dir/subdir should exist")
				assert.True(t, info.IsDir(), "dir/subdir should be a directory")

				// Verify file was created
				filePath := filepath.Join(extractDir, "dir/file.txt")
				content, err := os.ReadFile(filePath)
				assert.NoError(t, err)
				assert.Equal(t, "content", string(content))
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Create ZIP file with specified files
			buf := new(bytes.Buffer)
			zipWriter := zip.NewWriter(buf)

			for fileName, content := range tc.files {
				// Handle directory entries (names ending with /)
				if strings.HasSuffix(fileName, "/") {
					header := &zip.FileHeader{
						Name:   fileName,
						Method: zip.Deflate,
					}
					header.SetMode(0755 | os.ModeDir)
					_, err := zipWriter.CreateHeader(header)
					require.NoError(t, err)
				} else {
					w, err := zipWriter.Create(fileName)
					require.NoError(t, err)
					_, err = w.Write([]byte(content))
					require.NoError(t, err)
				}
			}

			err := zipWriter.Close()
			require.NoError(t, err)

			// Extract the ZIP
			tmpDir := t.TempDir()
			reader := bytes.NewReader(buf.Bytes())

			log := &testLogger{t: t}
			err = extractZipToFolder(reader, int64(buf.Len()), "test", tmpDir, log)
			assert.NoError(t, err)

			// Run test-specific validation
			tc.validate(t, tmpDir, tc.files)
		})
	}
}

// TestExtractZipToFolder_PreventPathTraversal verifies that malicious ZIP files
// with path traversal attempts are blocked by os.Root security
func TestExtractZipToFolder_PreventPathTraversal(t *testing.T) {
	// Create temp directory first to calculate its depth
	extractDir := t.TempDir()

	// Calculate directory depth (number of path separators)
	pathDepth := strings.Count(extractDir, string(filepath.Separator))
	// Add extra levels to ensure we escape (e.g., +2 to go above tmpDir)
	escapeDepth := pathDepth + 2

	// Build traversal strings based on actual depth
	escapeToRoot := strings.Repeat("../", escapeDepth)
	escapeToParent := strings.Repeat("../", pathDepth)

	// Create a malicious ZIP file with path traversal attempts
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	maliciousFiles := map[string]string{
		// Escape to filesystem root, then target system directories
		escapeToRoot + "etc/evil.txt":  "should not escape to /etc",
		escapeToRoot + "tmp/evil.txt":  "should not escape to /tmp",
		escapeToRoot + "root/evil.txt": "should not escape to /root",

		// Escape to parent of tmpDir
		escapeToParent + "evil.txt": "should not escape to parent",

		// Mixed traversal
		"subdir/" + escapeToParent + "bad.txt": "should not escape via subdir",

		// Absolute paths (always blocked)
		"/etc/absolute.txt": "absolute path attack",

		// Safe file
		"normal.txt": "this is fine",
	}

	for name, content := range maliciousFiles {
		w, err := zipWriter.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}

	err := zipWriter.Close()
	require.NoError(t, err)

	// Extract - malicious paths should be blocked by os.Root
	reader := bytes.NewReader(buf.Bytes())
	log := &testLogger{t: t}

	err = extractZipToFolder(reader, int64(buf.Len()), "test", extractDir, log)

	// With os.Root, extraction should fail when encountering path traversal attempts
	assert.Error(t, err, "Extraction should fail when malicious paths are encountered")
	assert.Contains(t, err.Error(), "path escapes from parent", "Error should indicate path traversal was blocked")

	// Defense-in-depth: verify that no malicious files were created outside extractDir
	// Even though extraction failed, ensure no partial writes escaped
	parentDir := filepath.Dir(extractDir)
	grandParentDir := filepath.Dir(parentDir)

	checkPaths := []string{
		filepath.Join(parentDir, "evil.txt"),
		filepath.Join(parentDir, "bad.txt"),
		filepath.Join(grandParentDir, "evil.txt"),
		"/etc/evil.txt",
		"/tmp/evil.txt",
		"/root/evil.txt",
		"/etc/absolute.txt",
	}

	for _, path := range checkPaths {
		_, err := os.Stat(path)
		// Expect "no such file or directory" - meaning the file wasn't created
		assert.ErrorIs(t, err, fs.ErrNotExist, "Malicious file should not exist at %s", path)
	}
}
