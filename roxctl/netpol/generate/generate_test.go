package generate

import (
	goerrors "errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
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
		expectedSynthError    string
		expectedSynthWarning  string
		expectedValidateError error
		stopOnFirstErr        bool
		outFile               string
		outDir                string
		removeOutputPath      bool
	}{
		"not existing inputFolderPath should raise 'does not exist' error": {
			inputFolderPath:    "/tmp/xxx",
			expectedSynthError: "the path \"/tmp/xxx\" does not exist",
		},
		"happyPath": {
			inputFolderPath:    "testdata/minimal",
			expectedSynthError: "",
		},
		"empty yamls should yield error no kubernetes resources found": {
			inputFolderPath:      "testdata/empty-yamls",
			expectedSynthError:   "could not find any Kubernetes workload resources",
			expectedSynthWarning: "Object 'Kind' is missing in",
		},
		"stopOnFirstError": {
			inputFolderPath:      "testdata/dirty",
			expectedSynthError:   "could not find any Kubernetes workload resources",
			expectedSynthWarning: "error parsing",
			stopOnFirstErr:       true,
		},
		"output should be written to a single file": {
			inputFolderPath:    "testdata/minimal",
			expectedSynthError: "",
			outFile:            d.T().TempDir() + "/out.yaml",
			outDir:             "",
			removeOutputPath:   false,
		},
		"output should be written to files in a directory": {
			inputFolderPath:    "testdata/minimal",
			expectedSynthError: "",
			outFile:            "",
			outDir:             d.T().TempDir(),
			removeOutputPath:   true,
		},
		"should return error that the dir already exists": {
			inputFolderPath:       "testdata/minimal",
			expectedValidateError: errox.AlreadyExists,
			expectedSynthError:    "",
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
					TreatWarningsAsErrors: false, // this is tested in netpol/resources/npg_test.go
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

			warns, errs := generateNetpolCmd.generateNetpol(generator)
			if tt.expectedSynthError != "" {
				d.Require().Error(goerrors.Join(errs...))
				d.Assert().ErrorContains(goerrors.Join(errs...), tt.expectedSynthError)
			} else {
				d.Assert().NoError(goerrors.Join(errs...))
			}
			if tt.expectedSynthWarning != "" {
				d.Require().Error(goerrors.Join(warns...))
				d.Assert().ErrorContains(goerrors.Join(warns...), tt.expectedSynthWarning)
			} else {
				d.Assert().NoError(goerrors.Join(warns...))
			}
		})
	}
}
