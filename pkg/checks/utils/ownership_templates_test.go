// +build linux
// +build cgo

package utils

import (
	"io/ioutil"
	"os"
	"os/user"
	"strconv"
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getInt(val string) int {
	v, _ := strconv.Atoi(val)
	return v
}

func setOwnership(file, u, g string) error {
	us, err := user.Lookup(u)
	if err != nil {
		return err
	}
	group, err := user.LookupGroup(g)
	if err != nil {
		return err
	}
	err = os.Chown(file, getInt(group.Gid), getInt(us.Uid))
	return err
}

// Returns file name with these ownership settings
func createTestFileOwnership(u, g string) (string, error) {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}
	if err := setOwnership(file.Name(), u, g); err != nil {
		return "", err
	}
	return file.Name(), nil
}

// Returns file name with these ownership settings
func createTestDirOwnership(u, g string) (dir string, fileA string, fileB string, err error) {
	dir, a, b, err := createTestDir()
	if err = setOwnership(a, u, g); err != nil {
		return
	}
	if err = setOwnership(b, u, g); err != nil {
		return
	}
	return
}

func TestCompareFileOwnership(t *testing.T) {
	ContainerPathPrefix = ""
	currentUser, err := user.Current()
	require.Nil(t, err)

	// Match the ownership
	expectedResult := v1.CheckResult{Result: v1.CheckStatus_PASS}
	f, err := createTestFileOwnership(currentUser.Username, currentUser.Username)
	require.Nil(t, err)
	result := compareFileOwnership(f, currentUser.Username, currentUser.Username)
	assert.Equal(t, expectedResult.Result, result.Result)
	assert.Equal(t, 0, len(result.Notes))

	// Testing for this is hard because of the rules of user:group. The base user requires sudo to chown
	// Compare to a file that doesn't have the right ownership
	// Match the user but not the group
	expectedResult = v1.CheckResult{Result: v1.CheckStatus_WARN}
	result = compareFileOwnership("/etc/passwd", currentUser.Username, currentUser.Username)
	result = compareFileOwnership(f, currentUser.Username, "docker")
	assert.Equal(t, expectedResult.Result, result.Result)
	assert.Equal(t, 1, len(result.Notes))
}

func TestFileOwnershipCheck(t *testing.T) {
	ContainerPathPrefix = ""
	// Set up file to check against
	currentUser, err := user.Current()
	require.Nil(t, err)
	expectedResult := v1.CheckResult{Result: v1.CheckStatus_PASS}
	f, err := createTestFileOwnership(currentUser.Username, currentUser.Username)
	require.Nil(t, err)

	benchmark := NewOwnershipCheck("Test bench", "desc", f, currentUser.Username, currentUser.Username)
	result := benchmark.Run()
	assert.Equal(t, expectedResult.Result, result.Result)
	assert.Equal(t, 0, len(result.Notes))

	// Check empty file
	expectedResult = v1.CheckResult{Result: v1.CheckStatus_NOTE}
	benchmark = NewOwnershipCheck("Test bench", "desc", "", currentUser.Username, currentUser.Username)
	result = benchmark.Run()
	assert.Equal(t, expectedResult.Result, result.Result)
	assert.Equal(t, 1, len(result.Notes))
}

func TestRecursiveOwnershipCheck(t *testing.T) {
	ContainerPathPrefix = ""
	currentUser, err := user.Current()
	require.Nil(t, err)
	dir, _, _, err := createTestDirOwnership(currentUser.Username, currentUser.Username)
	require.Nil(t, err)

	expectedResult := v1.CheckResult{Result: v1.CheckStatus_PASS}
	benchmark := NewRecursiveOwnershipCheck("test bench", "desc", dir, currentUser.Username, currentUser.Username)
	result := benchmark.Run()
	assert.Equal(t, expectedResult.Result, result.Result)
	assert.Equal(t, 0, len(result.Notes))

	// Check empty file
	expectedResult = v1.CheckResult{Result: v1.CheckStatus_NOTE}
	benchmark = NewRecursiveOwnershipCheck("Test bench", "desc", "", currentUser.Username, currentUser.Username)
	result = benchmark.Run()
	assert.Equal(t, expectedResult.Result, result.Result)
	assert.Equal(t, 1, len(result.Notes))
}
