package diff

import (
	"os"
	"path"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/common/npg"
	"github.com/stretchr/testify/suite"
)

func TestDiffAnalyzeNetpolCommand(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(diffAnalyzeNetpolTestSuite))
}

type diffAnalyzeNetpolTestSuite struct {
	suite.Suite
}

func (d *diffAnalyzeNetpolTestSuite) TestValidDiffCommand() {
	tmpOutFileName := d.T().TempDir() + "/out"
	outFileTxt := tmpOutFileName + ".txt"
	cases := []struct {
		name                  string
		inputFolderPath1      string
		inputFolderPath2      string
		outFile               string
		runCommand            bool
		removeOutputPath      bool
		expectedValidateError error
	}{
		{
			name:                  "Empty dir1 input should return validation error",
			inputFolderPath1:      "",
			inputFolderPath2:      "",
			expectedValidateError: errox.InvalidArgs,
		},
		{
			name:                  "Empty dir2 should return validation error",
			inputFolderPath1:      "/dev/null",
			inputFolderPath2:      "",
			expectedValidateError: errox.InvalidArgs,
		},
		{
			name:                  "Valid inputs should write output to a provided output file",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			expectedValidateError: nil,
			outFile:               outFileTxt,
			runCommand:            true, // to write the output - critic for next tests
			removeOutputPath:      false,
		},
		{
			name:                  "Existing output file input without using remove flag should return error that the file already exists",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			expectedValidateError: errox.AlreadyExists,
			outFile:               outFileTxt,
			removeOutputPath:      false,
		},
		{
			name:                  "Existing output file input with using remove flag should override existing file",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			expectedValidateError: nil,
			runCommand:            true,
			outFile:               outFileTxt,
			removeOutputPath:      true,
		},
	}

	for _, tt := range cases {
		tt := tt
		d.Run(tt.name, func() {
			env, _, _ := mocks.NewEnvWithConn(nil, d.T())
			diffNetpolCmd := diffNetpolCommand{
				inputFolderPath1: tt.inputFolderPath1,
				inputFolderPath2: tt.inputFolderPath2,
				outputFilePath:   tt.outFile,
				removeOutputPath: tt.removeOutputPath,
				env:              env,
			}

			analyzer, err := diffNetpolCmd.construct()
			d.NoError(err)

			err = diffNetpolCmd.validate()
			if tt.expectedValidateError != nil {
				d.Require().Error(err)
				d.ErrorIs(err, tt.expectedValidateError)
				return
			}
			d.NoError(err)

			if tt.runCommand {
				err := diffNetpolCmd.analyzeConnectivityDiff(analyzer)
				d.NoError(err)
			}

			if tt.outFile != "" {
				_, err := os.Stat(tt.outFile)
				d.NoError(err) // out file should exist
			}
		})
	}
}

func (d *diffAnalyzeNetpolTestSuite) TestAnalyzeDiffCommand() {
	cases := []struct {
		name                     string
		inputFolderPath1         string
		inputFolderPath2         string
		strict                   bool
		stopOnFirstErr           bool
		outputToFile             bool
		outputFormat             string
		expectedAnalysisError    error
		expectedErrorMsgContains string
	}{
		{
			name:                  "Not existing input folder paths should result in error 'os.ErrNotExist'",
			inputFolderPath1:      "/tmp/xxx",
			inputFolderPath2:      "/tmp/xxx",
			expectedAnalysisError: os.ErrNotExist,
		},
		{
			name:                  "Inputs with no resources should result in general NP-Guard error",
			inputFolderPath1:      "testdata/empty-yamls",
			inputFolderPath2:      "testdata/empty-yamls",
			expectedAnalysisError: npg.ErrErrors,
		},
		{
			name:                  "Treating warnings as errors should result in error of type 'npg.ErrWarnings'",
			inputFolderPath1:      "testdata/acs-zeroday-with-invalid-doc",
			inputFolderPath2:      "testdata/acs-zeroday-with-invalid-doc",
			expectedAnalysisError: npg.ErrWarnings,
			strict:                true,
		},
		{
			name:                  "Warnings on invalid input docs without using strict flag should not indicate warnings as errors",
			inputFolderPath1:      "testdata/acs-zeroday-with-invalid-doc",
			inputFolderPath2:      "testdata/acs-zeroday-with-invalid-doc",
			expectedAnalysisError: nil,
		},
		{
			name:                  "Stop on first error with malformed yaml inputs should stop with general NP-Guard error",
			inputFolderPath1:      "testdata/dirty", // yaml document malformed
			inputFolderPath2:      "testdata/dirty",
			expectedAnalysisError: npg.ErrErrors,
			stopOnFirstErr:        true,
		},
		{
			name:                     "Not supported output format should result an error in formatting connectivity diff",
			inputFolderPath1:         "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:         "testdata/netpol-diff-example-minimal",
			outputFormat:             "docx",
			expectedErrorMsgContains: "docx output format is not supported.",
		},
		{
			name:                  "Testing Diff between two dirs should run successfully without errors",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			expectedAnalysisError: nil,
		},

		{
			name:                  "Testing Diff between two dirs output should be written to default txt output file",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			expectedAnalysisError: nil,
			outputToFile:          true,
			outputFormat:          defaultOutputFormat,
		},
		{
			name:                  "Testing Diff between two dirs output should be written to default md output file",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			expectedAnalysisError: nil,
			outputToFile:          true,
			outputFormat:          "md",
		},
		{
			name:                  "Testing Diff between two dirs output should be written to default csv output file",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			expectedAnalysisError: nil,
			outputToFile:          true,
			outputFormat:          "csv",
		},
		{
			name:                  "ACS example output should be written to default txt output file",
			inputFolderPath1:      "testdata/acs-security-demos",
			inputFolderPath2:      "testdata/acs-security-demos-new-version",
			expectedAnalysisError: nil,
			outputToFile:          true,
			outputFormat:          "txt",
		},
		{
			name:                  "ACS example output should be written to default md output file",
			inputFolderPath1:      "testdata/acs-security-demos",
			inputFolderPath2:      "testdata/acs-security-demos-new-version",
			expectedAnalysisError: nil,
			outputToFile:          true,
			outputFormat:          "md",
		},
		{
			name:                  "ACS example output should be written to default csv output file",
			inputFolderPath1:      "testdata/acs-security-demos",
			inputFolderPath2:      "testdata/acs-security-demos-new-version",
			expectedAnalysisError: nil,
			outputToFile:          true,
			outputFormat:          "csv",
		},
	}
	for _, tt := range cases {
		tt := tt
		d.Run(tt.name, func() {
			env, _, _ := mocks.NewEnvWithConn(nil, d.T())
			diffNetpolCmd := diffNetpolCommand{
				stopOnFirstError:      tt.stopOnFirstErr,
				treatWarningsAsErrors: tt.strict,
				inputFolderPath1:      tt.inputFolderPath1,
				inputFolderPath2:      tt.inputFolderPath2,
				outputToFile:          tt.outputToFile,
				outputFormat:          tt.outputFormat,
				env:                   env,
			}

			analyzer, err := diffNetpolCmd.construct()
			d.NoError(err)

			err = diffNetpolCmd.validate()
			d.NoError(err)

			err = diffNetpolCmd.analyzeConnectivityDiff(analyzer)
			if tt.expectedAnalysisError != nil {
				d.Require().Error(err)
				d.ErrorIs(err, tt.expectedAnalysisError)
				return
			}
			if tt.expectedErrorMsgContains != "" {
				d.Require().Error(err)
				d.ErrorContains(err, tt.expectedErrorMsgContains)
				return
			}
			d.NoError(err)

			if tt.outputToFile {
				defaultFile := diffNetpolCmd.getDefaultFileName()
				formatSuffix := defaultOutputFormat
				if tt.outputFormat != "" {
					d.Contains(defaultFile, tt.outputFormat)
					formatSuffix = tt.outputFormat
				}
				output, err := os.ReadFile(defaultFile)
				d.NoError(err)

				expectedOutput, err := os.ReadFile(path.Join(tt.inputFolderPath2, "diff_output."+formatSuffix))
				d.NoError(err)
				d.Equal(string(expectedOutput), string(output))

				d.NoError(os.Remove(defaultFile))
			}
		})
	}

}
