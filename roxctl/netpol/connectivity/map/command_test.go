package connectivitymap

import (
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stretchr/testify/suite"
)

func TestConnectivityMapCommand(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(connectivityMapCommandSuite))
}

type connectivityMapCommandSuite struct {
	suite.Suite
}

func (d *connectivityMapCommandSuite) TestValidate() {
	cases := []struct {
		name               string
		inputFolderPath    string
		expectedErr        error
		writeDummyToOutput bool
		removeOutputPath   bool
	}{
		{
			name:             "output should be written to a single file",
			inputFolderPath:  "testdata/minimal",
			expectedErr:      nil,
			removeOutputPath: false,
		},
		{
			name:               "should return error that the file already exists",
			inputFolderPath:    "testdata/minimal",
			expectedErr:        errox.AlreadyExists,
			writeDummyToOutput: true,
			removeOutputPath:   false,
		},
		{
			name:             "should override existing file",
			inputFolderPath:  "testdata/minimal",
			expectedErr:      nil,
			removeOutputPath: true,
		},
	}

	for _, tt := range cases {
		d.Run(tt.name, func() {
			env, _, _ := mocks.NewEnvWithConn(nil, d.T())
			analyzeNetpolCmd := Cmd{
				stopOnFirstError:      false,
				treatWarningsAsErrors: false,
				inputFolderPath:       tt.inputFolderPath,
				outputFilePath:        d.T().TempDir() + "/out.txt",
				removeOutputPath:      tt.removeOutputPath,
				outputToFile:          false,
				focusWorkload:         "",
				outputFormat:          "",
				env:                   env,
			}

			if tt.writeDummyToOutput {
				d.Require().NoError(os.WriteFile(analyzeNetpolCmd.outputFilePath, []byte("hello\n"), 0644))
			}

			err := analyzeNetpolCmd.validate()
			if tt.expectedErr != nil {
				d.Require().Error(err)
				d.Assert().ErrorIs(err, tt.expectedErr)
				return
			}
			d.Assert().NoError(err)
		})
	}
}
