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
		stopOnFirstErr        bool
		treatWarningsAsErrors bool
		outFile               string
		outDir                string
		removeOutputPath      bool

		expectedValidationError error
		expectedWarnings        []string
		expectedErrors          []string
	}{
		"not existing inputFolderPath should raise 'does not exist' error": {
			inputFolderPath: "/tmp/xxx",

			expectedValidationError: nil,
			expectedWarnings:        []string{},
			expectedErrors: []string{
				"the path \"/tmp/xxx\" does not exist",
				"error generating network policies: could not find any Kubernetes workload resources",
			},
		},
		"happyPath": {
			inputFolderPath:         "testdata/minimal",
			expectedValidationError: nil,
			expectedWarnings:        []string{},
			expectedErrors:          []string{},
		},
		"empty yamls should yield error no kubernetes resources found": {
			inputFolderPath:         "testdata/empty-yamls",
			expectedValidationError: nil,
			expectedWarnings: []string{
				"unable to decode \"testdata/empty-yamls/empty.yaml\": Object 'Kind' is missing in",
				"unable to decode \"testdata/empty-yamls/empty2.yaml\": Object 'Kind' is missing in"},
			expectedErrors: []string{"could not find any Kubernetes workload resources"},
		},
		"generation should stop on first warning when warnings are treated as errors ": {
			inputFolderPath:         "testdata/dirty",
			stopOnFirstErr:          true,
			treatWarningsAsErrors:   true,
			expectedValidationError: nil,
			expectedErrors:          []string{"could not find any Kubernetes workload resources"},
			expectedWarnings: []string{
				"error parsing testdata/dirty/backend.yaml",
				"error parsing testdata/dirty/frontend.yaml",
			},
		},
		"output should be written to a single file": {
			inputFolderPath:         "testdata/minimal",
			outFile:                 d.T().TempDir() + "/out.yaml",
			outDir:                  "",
			removeOutputPath:        false,
			expectedValidationError: nil,
			expectedWarnings:        []string{},
			expectedErrors:          []string{},
		},
		"output should be written to files in a directory": {
			inputFolderPath:         "testdata/minimal",
			outFile:                 "",
			outDir:                  d.T().TempDir(),
			removeOutputPath:        true,
			expectedValidationError: nil,
			expectedWarnings:        []string{},
			expectedErrors:          []string{},
		},
		"should return error that the dir already exists": {
			inputFolderPath:  "testdata/minimal",
			outFile:          "",
			outDir:           d.T().TempDir(),
			removeOutputPath: false,

			expectedValidationError: errox.AlreadyExists,
			expectedWarnings:        []string{},
			expectedErrors:          []string{},
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
			if tt.expectedValidationError != nil {
				d.Require().Error(err, "validation error is expected")
				d.Assert().ErrorIs(err, tt.expectedValidationError)
				return
			}
			d.Assert().NoError(err)

			warns, errs := generateNetpolCmd.generateNetpol(generator)
			d.Require().Lenf(errs, len(tt.expectedErrors), "number of errors should be %d", len(tt.expectedErrors))
			d.Require().Lenf(warns, len(tt.expectedWarnings), "number of warnings should be %d", len(tt.expectedWarnings))

			for _, expError := range tt.expectedErrors {
				if expError != "" {
					d.Require().Error(goerrors.Join(errs...))
					d.Assert().ErrorContainsf(goerrors.Join(errs...), expError,
						"Expected errors to contain %s", tt.expectedErrors)
				} else {
					d.Assert().NoError(goerrors.Join(errs...))
				}
			}
			for _, expWarn := range tt.expectedWarnings {
				if expWarn != "" {
					d.Require().Error(goerrors.Join(warns...))
					d.Assert().ErrorContainsf(goerrors.Join(warns...), expWarn,
						"Expected warnings to contain %s", tt.expectedWarnings)
				} else {
					d.Assert().NoError(goerrors.Join(warns...))
				}
			}
		})
	}
}
