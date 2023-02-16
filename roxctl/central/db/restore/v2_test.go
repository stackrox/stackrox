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
	t.Parallel()
	testDir := t.TempDir()
	testFile := path.Join(testDir, "test-file")
	_, err := os.Create(testFile)
	require.NoError(t, err)

	cases := map[string]struct {
		cmd *centralDbRestoreCommand
		err error
	}{
		"if file is unset, expect an InvalidArgs error": {
			cmd: &centralDbRestoreCommand{},
			err: errox.InvalidArgs,
		},
		"if file is set, but does not exist, expect an NotFound error": {
			cmd: &centralDbRestoreCommand{file: "non-existent-file"},
			err: errox.NotFound,
		},
		"if file is set, but is a directory, expect an InvalidArgs error": {
			cmd: &centralDbRestoreCommand{file: testDir},
			err: errox.InvalidArgs,
		},
		"if file is set, and  exists, no error should be returned": {
			cmd: &centralDbRestoreCommand{file: testFile},
		},
	}
	for name, c := range cases {
		tc := c
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := tc.cmd.validate()
			if tc.err != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
