package fileutils

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMkdirAllInRoot(t *testing.T) {
	tmpDir := t.TempDir()
	root, err := os.OpenRoot(tmpDir)
	require.NoError(t, err)
	defer utils.IgnoreError(root.Close)

	// Test creating nested directories
	err = MkdirAllInRoot(root, "a/b/c/d", 0755)
	assert.NoError(t, err)

	// Verify directories were created
	_, err = os.Stat(filepath.Join(tmpDir, "a/b/c/d"))
	assert.NoError(t, err)

	// Test idempotency - calling again should succeed
	err = MkdirAllInRoot(root, "a/b/c/d", 0755)
	assert.NoError(t, err)

	// Test creating sibling directory
	err = MkdirAllInRoot(root, "a/b/e", 0755)
	assert.NoError(t, err)

	_, err = os.Stat(filepath.Join(tmpDir, "a/b/e"))
	assert.NoError(t, err)
}

func TestMkdirAllInRoot_EmptyPath(t *testing.T) {
	tmpDir := t.TempDir()
	root, err := os.OpenRoot(tmpDir)
	require.NoError(t, err)
	defer utils.IgnoreError(root.Close)

	// Test with empty path
	err = MkdirAllInRoot(root, "", 0755)
	assert.NoError(t, err)

	// Test with current directory
	err = MkdirAllInRoot(root, ".", 0755)
	assert.NoError(t, err)
}

func TestMkdirAllInRoot_SingleLevel(t *testing.T) {
	tmpDir := t.TempDir()
	root, err := os.OpenRoot(tmpDir)
	require.NoError(t, err)
	defer utils.IgnoreError(root.Close)

	// Test creating single-level directory
	err = MkdirAllInRoot(root, "single", 0755)
	assert.NoError(t, err)

	_, err = os.Stat(filepath.Join(tmpDir, "single"))
	assert.NoError(t, err)
}

func TestMkdirAllInRoot_PathTraversalProtection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Logf("using tmpDir: %s", tmpDir)
	root, err := os.OpenRoot(tmpDir)
	require.NoError(t, err)
	defer utils.IgnoreError(root.Close)

	testCases := map[string]struct {
		maliciousPath string
		expectError   bool
		errorContains string
	}{
		"should block parent directory traversal": {
			maliciousPath: "../../etc",
			expectError:   true,
			errorContains: "path escapes from parent",
		},
		"should block deep parent traversal": {
			maliciousPath: "../../../tmp",
			expectError:   true,
			errorContains: "path escapes from parent",
		},
		"should block absolute path traversal": {
			maliciousPath: "/../root",
			expectError:   true,
			errorContains: "path escapes from parent",
		},
		"should block mixed relative and parent traversal": {
			maliciousPath: "subdir/../../../etc",
			expectError:   true,
			errorContains: "path escapes from parent",
		},
		"should block absolute paths to system directories": {
			maliciousPath: "/etc/evil",
			expectError:   true,
			errorContains: "path escapes from parent",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Call MkdirAllInRoot - os.Root should block path traversal attempts
			err := MkdirAllInRoot(root, tc.maliciousPath, 0755)

			if tc.expectError {
				// os.Root should return an error for path traversal attempts
				assert.Errorf(t, err, "Expected error for malicious path %q", tc.maliciousPath)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains,
						"Error should indicate path traversal was blocked")
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWriteFileInRoot(t *testing.T) {
	tmpDir := t.TempDir()
	root, err := os.OpenRoot(tmpDir)
	require.NoError(t, err)
	defer utils.IgnoreError(root.Close)

	testCases := map[string]struct {
		fileName string
		content  string
		perm     os.FileMode
	}{
		"should write simple file": {
			fileName: "test.txt",
			content:  "test content",
			perm:     0644,
		},
		"should write file in nested directory": {
			fileName: "a/b/c/nested.txt",
			content:  "nested content",
			perm:     0644,
		},
		"should write file with executable permissions": {
			fileName: "script.sh",
			content:  "#!/bin/bash\necho test",
			perm:     0755,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			rc := io.NopCloser(strings.NewReader(tc.content))
			err := WriteFileInRoot(root, tc.fileName, tc.perm, rc)
			assert.NoError(t, err)

			// Verify file was created with correct content
			filePath := filepath.Join(tmpDir, tc.fileName)
			content, err := os.ReadFile(filePath)
			assert.NoError(t, err)
			assert.Equal(t, tc.content, string(content))

			// Verify permissions
			info, err := os.Stat(filePath)
			assert.NoError(t, err)
			assert.Equal(t, tc.perm, info.Mode().Perm())
		})
	}
}

func TestWriteFileInRoot_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	root, err := os.OpenRoot(tmpDir)
	require.NoError(t, err)
	defer utils.IgnoreError(root.Close)

	testCases := map[string]struct {
		maliciousPath string
		errorContains string
	}{
		"should block parent directory traversal": {
			maliciousPath: "../../etc/evil.txt",
			errorContains: "path escapes from parent",
		},
		"should block absolute path": {
			maliciousPath: "/etc/evil.txt",
			errorContains: "path escapes from parent",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			rc := io.NopCloser(strings.NewReader("malicious content"))
			err := WriteFileInRoot(root, tc.maliciousPath, 0644, rc)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.errorContains)
		})
	}
}
