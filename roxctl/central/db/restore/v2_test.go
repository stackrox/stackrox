package restore

import (
	"os"
	"path"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCentralDBRestore_Validate(t *testing.T) {
	cmd := &centralDbRestoreCommand{}

	// 1. If file is unset, expect an InvalidArgs error.
	err := cmd.validate()
	assert.Error(t, err)
	assert.ErrorIs(t, err, errox.InvalidArgs)

	// 2. If file is set, but does not exist, expect an NotFound error.
	cmd.file = "some-non-existent-file"
	err = cmd.validate()
	assert.Error(t, err)
	assert.ErrorIs(t, err, errox.NotFound)

	// 3. If file is set, and exists, no error should be returned.
	tmpDir := t.TempDir()
	testFile := path.Join(tmpDir, "test-file")
	_, err = os.Create(testFile)
	require.NoError(t, err)

	cmd.file = testFile
	err = cmd.validate()
	assert.NoError(t, err)
}
