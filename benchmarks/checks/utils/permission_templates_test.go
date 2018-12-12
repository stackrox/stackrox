package utils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Returns file name with these ownership settings
func createTestFile() (string, error) {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}
	return file.Name(), nil
}

func createTestDir() (dir string, fileA string, fileB string, err error) {
	dir, err = ioutil.TempDir("", "")
	if err != nil {
		return
	}
	fileA = filepath.Join(dir, "a.txt")
	if err = ioutil.WriteFile(fileA, []byte("hello world"), 0777); err != nil {
		return
	}
	fileB = filepath.Join(dir, "b.txt")
	if err = ioutil.WriteFile(fileB, []byte("hello world"), 0777); err != nil {
		return
	}
	return
}

func TestCompareFilePermissions(t *testing.T) {
	ContainerPathPrefix = ""

	// Test file not existing
	file := "/tmp/idontexist"
	expectedResult := storage.BenchmarkCheckResult{Result: storage.BenchmarkCheckStatus_NOTE}
	result := compareFilePermissions(file, 0777, true)
	assert.Equal(t, expectedResult.Result, result.Result)
	assert.Equal(t, 1, len(result.Notes))

	file, err := createTestFile()
	require.Nil(t, err)

	// Check equality
	err = os.Chmod(file, 0777)
	require.Nil(t, err)
	expectedResult = storage.BenchmarkCheckResult{Result: storage.BenchmarkCheckStatus_PASS}
	result = compareFilePermissions(file, 0777, true)
	assert.Equal(t, expectedResult, result)

	// Check less than with includesLower: true
	err = os.Chmod(file, 0666)
	require.Nil(t, err)
	expectedResult = storage.BenchmarkCheckResult{Result: storage.BenchmarkCheckStatus_PASS}
	result = compareFilePermissions(file, 0777, true)
	assert.Equal(t, expectedResult, result)

	// Check less than with includesLower: false
	err = os.Chmod(file, 0666)
	require.Nil(t, err)
	expectedResult = storage.BenchmarkCheckResult{Result: storage.BenchmarkCheckStatus_WARN}
	result = compareFilePermissions(file, 0777, false)
	assert.Equal(t, expectedResult.Result, result.Result)
	assert.Equal(t, 1, len(result.Notes))

	// Check permissions too high
	// Check less than with includesLower: false
	err = os.Chmod(file, 0777)
	require.Nil(t, err)
	expectedResult = storage.BenchmarkCheckResult{Result: storage.BenchmarkCheckStatus_WARN}
	result = compareFilePermissions(file, 0666, false)
	assert.Equal(t, expectedResult.Result, result.Result)
	assert.Equal(t, 1, len(result.Notes))
}

func TestPermissionsCheck(t *testing.T) {
	ContainerPathPrefix = ""

	// Test empty file
	expectedResult := storage.BenchmarkCheckResult{Result: storage.BenchmarkCheckStatus_NOTE}
	benchmark := NewPermissionsCheck("bench", "desc", "", 0777, true)
	result := benchmark.Run()
	assert.Equal(t, expectedResult.Result, result.Result)
	assert.Equal(t, 1, len(result.Notes))

	// Happy path
	file, err := createTestFile()
	require.Nil(t, err)
	err = os.Chmod(file, 0777)
	require.Nil(t, err)
	expectedResult = storage.BenchmarkCheckResult{Result: storage.BenchmarkCheckStatus_PASS}
	benchmark = NewPermissionsCheck("bench", "desc", file, 0777, true)
	result = benchmark.Run()
	assert.Equal(t, expectedResult, result)
}

func TestRecursivePermissionsCheck(t *testing.T) {
	ContainerPathPrefix = ""
	// Test empty file
	expectedResult := storage.BenchmarkCheckResult{Result: storage.BenchmarkCheckStatus_NOTE}
	benchmark := NewRecursivePermissionsCheck("bench", "desc", "", 0777, true)
	result := benchmark.Run()
	assert.Equal(t, expectedResult.Result, result.Result)
	assert.Equal(t, 1, len(result.Notes))

	// Generate directory with files
	dir, fileA, fileB, err := createTestDir()
	require.Nil(t, err)
	err = os.Chmod(fileA, 0666)
	require.Nil(t, err)
	err = os.Chmod(fileB, 0666)
	require.Nil(t, err)

	// Happy path
	expectedResult = storage.BenchmarkCheckResult{Result: storage.BenchmarkCheckStatus_PASS}
	benchmark = NewRecursivePermissionsCheck("bench", "desc", dir, 0666, true)
	result = benchmark.Run()
	assert.Equal(t, expectedResult, result)

	// One file has the wrong permissions
	err = os.Chmod(fileB, 0777)
	expectedResult = storage.BenchmarkCheckResult{Result: storage.BenchmarkCheckStatus_WARN}
	benchmark = NewRecursivePermissionsCheck("bench", "desc", dir, 0666, true)
	result = benchmark.Run()
	assert.Equal(t, expectedResult.Result, result.Result)
	assert.Equal(t, 1, len(result.Notes))
}
