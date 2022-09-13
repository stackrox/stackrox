package netpol

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/mocks"
	"github.com/stretchr/testify/suite"
)

func TestGenerateNetpolCommand(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(generateNetpolTestSuite))
}

type generateNetpolTestSuite struct {
	suite.Suite
	tmpOutFile string
	tmpOutDir  string
}

func (d *generateNetpolTestSuite) SetupTest() {
	d.tmpOutDir = d.T().TempDir()
	d.tmpOutFile = d.T().TempDir() + "/out.yaml"
}

func (d *generateNetpolTestSuite) TestGenerateNetpol() {
	testCmd := &cobra.Command{Use: "test"}
	testCmd.Flags().String("output-dir", "", "")
	testCmd.Flags().String("output-file", "", "")

	cases := map[string]struct {
		inputFolderPath        string
		expectedConstructError error
		expectedSynthError     error
		expectedValidateError  error
		strict                 bool
		stopOnFirstErr         bool
		outFile                string
		outDir                 string
		removeOutputPath       bool
	}{
		"not existing inputFolderPath should raise 'os.ErrNotExist' error": {
			inputFolderPath:        "/tmp/xxx",
			expectedConstructError: nil,
			expectedSynthError:     os.ErrNotExist,
		},
		"happyPath": {
			inputFolderPath:        "testdata/minimal",
			expectedConstructError: nil,
			expectedSynthError:     nil,
		},
		"treating warnings as errors": {
			inputFolderPath:        "testdata/empty-yamls",
			expectedConstructError: nil,
			expectedSynthError:     errNPGWarningsIndicator,
			strict:                 true,
		},
		"stopOnFistError": {
			inputFolderPath:        "testdata/dirty",
			expectedConstructError: nil,
			expectedSynthError:     errNPGErrorsIndicator,
			stopOnFirstErr:         true,
		},
		"output should be written to a single file": {
			inputFolderPath:        "testdata/minimal",
			expectedConstructError: nil,
			expectedSynthError:     nil,
			outFile:                d.tmpOutFile,
			removeOutputPath:       false,
		},
		"output should be written to files in a directory": {
			inputFolderPath:        "testdata/minimal",
			expectedConstructError: nil,
			expectedSynthError:     nil,
			outDir:                 d.tmpOutDir,
			removeOutputPath:       true,
		},
		"should return error that the dir already exists": {
			inputFolderPath:        "testdata/minimal",
			expectedConstructError: nil,
			expectedValidateError:  errox.AlreadyExists,
			expectedSynthError:     nil,
			outDir:                 d.tmpOutDir,
			removeOutputPath:       false,
		},
	}

	for name, tt := range cases {
		d.Run(name, func() {
			env, _, _ := mocks.NewEnvWithConn(nil, d.T())
			generateNetpolCmd := generateNetpolCommand{
				offline:               true,
				stopOnFirstError:      tt.stopOnFirstErr,
				treatWarningsAsErrors: tt.strict,
				inputFolderPath:       "", // set through construct
				outputFolderPath:      tt.outDir,
				outputFilePath:        tt.outFile,
				removeOutputPath:      tt.removeOutputPath,
				env:                   env,
				printer:               nil,
			}
			if tt.outDir != "" {
				d.Assert().NoError(testCmd.Flags().Set("output-dir", tt.outDir))
			}
			if tt.outFile != "" {
				d.Assert().NoError(testCmd.Flags().Set("output-file", tt.outFile))
			}

			generator, err := generateNetpolCmd.construct([]string{tt.inputFolderPath}, testCmd)
			if tt.expectedConstructError != nil {
				d.Require().Error(err)
				d.Assert().ErrorIs(err, tt.expectedConstructError)
			} else {
				d.Assert().NoError(err)
			}

			err = generateNetpolCmd.validate()
			if tt.expectedValidateError != nil {
				d.Require().Error(err)
				d.Assert().ErrorIs(err, tt.expectedValidateError)
			} else {
				d.Assert().NoError(err)
			}

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
