package configurationfiles

import (
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
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

func TestCompareFilePermissions(t *testing.T) {
	// Test file not existing
	file := "/tmp/idontexist"
	expectedResult := v1.CheckResult{Result: v1.CheckStatus_NOTE}
	result := compareFilePermissions(file, 0777, true)
	assert.Equal(t, expectedResult.Result, result.Result)
	assert.Equal(t, 1, len(result.Notes))

	file, err := createTestFile()
	require.Nil(t, err)

	// Check equality
	err = os.Chmod(file, 0777)
	require.Nil(t, err)
	expectedResult = v1.CheckResult{Result: v1.CheckStatus_PASS}
	result = compareFilePermissions(file, 0777, true)
	assert.Equal(t, expectedResult, result)

	// Check less than with includesLower: true
	err = os.Chmod(file, 0666)
	require.Nil(t, err)
	expectedResult = v1.CheckResult{Result: v1.CheckStatus_PASS}
	result = compareFilePermissions(file, 0777, true)
	assert.Equal(t, expectedResult, result)

	// Check less than with includesLower: false
	err = os.Chmod(file, 0666)
	require.Nil(t, err)
	expectedResult = v1.CheckResult{Result: v1.CheckStatus_WARN}
	result = compareFilePermissions(file, 0777, false)
	assert.Equal(t, expectedResult.Result, result.Result)
	assert.Equal(t, 1, len(result.Notes))

	// Check permissions too high
	// Check less than with includesLower: false
	err = os.Chmod(file, 0777)
	require.Nil(t, err)
	expectedResult = v1.CheckResult{Result: v1.CheckStatus_WARN}
	result = compareFilePermissions(file, 0666, false)
	assert.Equal(t, expectedResult.Result, result.Result)
	assert.Equal(t, 1, len(result.Notes))
}

func TestPermissionsCheck(t *testing.T) {
	// Test empty file
	expectedResult := v1.CheckResult{Result: v1.CheckStatus_NOTE}
	benchmark := newPermissionsCheck("bench", "desc", "", 0777, true)
	result := benchmark.Run()
	assert.Equal(t, expectedResult.Result, result.Result)
	assert.Equal(t, 1, len(result.Notes))

	// Happy path
	file, err := createTestFile()
	require.Nil(t, err)
	err = os.Chmod(file, 0777)
	require.Nil(t, err)
	expectedResult = v1.CheckResult{Result: v1.CheckStatus_PASS}
	benchmark = newPermissionsCheck("bench", "desc", file, 0777, true)
	result = benchmark.Run()
	assert.Equal(t, expectedResult, result)
}

func TestRecursivePermissionsCheck(t *testing.T) {
	// Test empty file
	expectedResult := v1.CheckResult{Result: v1.CheckStatus_NOTE}
	benchmark := newRecursivePermissionsCheck("bench", "desc", "", 0777, true)
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
	expectedResult = v1.CheckResult{Result: v1.CheckStatus_PASS}
	benchmark = newRecursivePermissionsCheck("bench", "desc", dir, 0666, true)
	result = benchmark.Run()
	assert.Equal(t, expectedResult, result)

	// One file has the wrong permissions
	err = os.Chmod(fileB, 0777)
	expectedResult = v1.CheckResult{Result: v1.CheckStatus_WARN}
	benchmark = newRecursivePermissionsCheck("bench", "desc", dir, 0666, true)
	result = benchmark.Run()
	assert.Equal(t, expectedResult.Result, result.Result)
	assert.Equal(t, 1, len(result.Notes))
}
