package generate

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/common/npg"
	"github.com/stretchr/testify/suite"
)

func TestGenerateNetpolCommand(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(generateNetpolTestSuite))
}

type generateNetpolTestSuite struct {
	suite.Suite
}

func (d *generateNetpolTestSuite) TestGenerateNetpol() {
	cases := map[string]struct {
		inputFolderPath       string
		expectedSynthError    error
		expectedValidateError error
		strict                bool
		stopOnFirstErr        bool
		outFile               string
		outDir                string
		removeOutputPath      bool
	}{
		"not existing inputFolderPath should raise 'os.ErrNotExist' error": {
			inputFolderPath:    "/tmp/xxx",
			expectedSynthError: os.ErrNotExist,
		},
		"happyPath": {
			inputFolderPath:    "testdata/minimal",
			expectedSynthError: nil,
		},
		"treating warnings as errors": {
			inputFolderPath:    "testdata/empty-yamls",
			expectedSynthError: npg.ErrWarnings,
			strict:             true,
		},
		"stopOnFistError": {
			inputFolderPath:    "testdata/dirty",
			expectedSynthError: npg.ErrErrors,
			stopOnFirstErr:     true,
		},
		"output should be written to a single file": {
			inputFolderPath:    "testdata/minimal",
			expectedSynthError: nil,
			outFile:            d.T().TempDir() + "/out.yaml",
			outDir:             "",
			removeOutputPath:   false,
		},
		"output should be written to files in a directory": {
			inputFolderPath:    "testdata/minimal",
			expectedSynthError: nil,
			outFile:            "",
			outDir:             d.T().TempDir(),
			removeOutputPath:   true,
		},
		"should return error that the dir already exists": {
			inputFolderPath:       "testdata/minimal",
			expectedValidateError: errox.AlreadyExists,
			expectedSynthError:    nil,
			outFile:               "",
			outDir:                d.T().TempDir(),
			removeOutputPath:      false,
		},
	}

	for name, tt := range cases {
		tt := tt
		d.Run(name, func() {
			testCmd := &cobra.Command{Use: "test"}
			testCmd.Flags().String("output-dir", "", "")
			testCmd.Flags().String("output-file", "", "")

			env, _, _ := mocks.NewEnvWithConn(nil, d.T())
			generateNetpolCmd := NetpolGenerateCmd{
				Options: NetpolGenerateOptions{
					StopOnFirstError:      tt.stopOnFirstErr,
					TreatWarningsAsErrors: tt.strict,
					OutputFolderPath:      tt.outDir,
					OutputFilePath:        tt.outFile,
					RemoveOutputPath:      tt.removeOutputPath,
				},
				offline:         true,
				inputFolderPath: "", // set through construct
				env:             env,
				printer:         nil,
			}
			if tt.outDir != "" {
				d.Assert().NoError(testCmd.Flags().Set("output-dir", tt.outDir))
			}
			if tt.outFile != "" {
				d.Assert().NoError(testCmd.Flags().Set("output-file", tt.outFile))
			}

			generator, err := generateNetpolCmd.construct([]string{tt.inputFolderPath}, testCmd)
			d.Assert().NoError(err)

			err = generateNetpolCmd.validate()
			if tt.expectedValidateError != nil {
				d.Require().Error(err)
				d.Assert().ErrorIs(err, tt.expectedValidateError)
				return
			}
			d.Assert().NoError(err)

			err = generateNetpolCmd.generateNetpol(generator)
			if tt.expectedSynthError != nil {
				d.Require().Error(err)
				d.Assert().ErrorIs(err, tt.expectedSynthError)
			} else {
				d.Assert().NoError(err)
			}
		})
	}
}
