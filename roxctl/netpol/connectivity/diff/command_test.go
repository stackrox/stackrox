package diff

import (
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		name                  string
		inputFolderPath1      string
		inputFolderPath2      string
		outFile               string
		expectedValidateError error
	}{
		{
			name:                  "Empty dir1 input should return validation error",
			inputFolderPath1:      "",
			inputFolderPath2:      "",
			outFile:               "",
			expectedValidateError: errox.InvalidArgs,
		},
		{
			name:                  "Empty dir2 should return validation error",
			inputFolderPath1:      "/dev/null",
			inputFolderPath2:      "",
			outFile:               "",
			expectedValidateError: errox.InvalidArgs,
		},
		{
			name:                  "Valid inputs should not raise any validate error",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			outFile:               "",
			expectedValidateError: nil,
		},
		{
			name:                  "Existing output file without using remove flag should return validate error that the file already exists",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			outFile:               "testdata/netpol-diff-example-minimal/diff_output.txt", // an existing file
			expectedValidateError: errox.AlreadyExists,
		},
		{
			name:                  "Non existing output file should not raise any validate error",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			outFile:               "testdata/netpol-diff-example-minimal/nonexisting", // an non-existing file
			expectedValidateError: nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			env, _, _ := mocks.NewEnvWithConn(nil, t)
			diffNetpolCmd := diffNetpolCommand{
				inputFolderPath1: tt.inputFolderPath1,
				inputFolderPath2: tt.inputFolderPath2,
				outputFilePath:   tt.outFile,
				env:              env,
			}

			_, err := diffNetpolCmd.construct()
			assert.NoError(t, err)

			err = diffNetpolCmd.validate()
			if tt.expectedValidateError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedValidateError)
				return
			}
			assert.NoError(t, err)
		})
	}
}
