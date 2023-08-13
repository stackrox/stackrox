package connectivitydiff

import (
	"errors"
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

func (d *diffAnalyzeNetpolTestSuite) TestDiffAnalyzeNetpol() {
	tmpOutFileName := d.T().TempDir() + "/out"
	outFileTxt := tmpOutFileName + ".txt"
	cases := []struct {
		name                  string
		inputFolderPath1      string
		inputFolderPath2      string
		expectedAnalysisError error
		expectedValidateError error
		strict                bool
		stopOnFirstErr        bool
		outFile               string
		outputToFile          bool
		outputFormat          string
		removeOutputPath      bool
		errStringContainment  bool
	}{
		{
			name:                  "not existing input folder paths error 'os.ErrNotExist'",
			inputFolderPath1:      "/tmp/xxx",
			inputFolderPath2:      "/tmp/xxx",
			expectedAnalysisError: os.ErrNotExist,
		},
		{
			name:                  "empty input paths errors with no resources found",
			inputFolderPath1:      "testdata/empty-yamls",
			inputFolderPath2:      "testdata/empty-yamls",
			expectedAnalysisError: npg.ErrErrors,
		},
		{
			name:                  "treating warnings as errors",
			inputFolderPath1:      "testdata/acs-zeroday-with-invalid-doc",
			inputFolderPath2:      "testdata/acs-zeroday-with-invalid-doc",
			expectedAnalysisError: npg.ErrWarnings,
			strict:                true,
		},
		{
			name:                  "warnings not indicated without strict",
			inputFolderPath1:      "testdata/acs-zeroday-with-invalid-doc",
			inputFolderPath2:      "testdata/acs-zeroday-with-invalid-doc",
			expectedAnalysisError: nil,
		},
		{
			name:                  "stop on first error",
			inputFolderPath1:      "testdata/dirty", // yaml document malformed
			inputFolderPath2:      "testdata/dirty",
			expectedAnalysisError: npg.ErrErrors,
			stopOnFirstErr:        true,
		},
		{
			name:                  "not supported output format",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			outputFormat:          "docx",
			errStringContainment:  true,
			expectedAnalysisError: errors.New("docx output format is not supported."),
		},
		{
			name:                  "happy path",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			expectedValidateError: nil,
			expectedAnalysisError: nil,
		},
		{
			name:                  "output should be written to a single file",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			expectedValidateError: nil,
			expectedAnalysisError: nil,
			outFile:               outFileTxt,
			removeOutputPath:      false,
		},
		{
			name:                  "should return error that the file already exists",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			expectedValidateError: errox.AlreadyExists,
			expectedAnalysisError: nil,
			outFile:               outFileTxt,
			removeOutputPath:      false,
		},
		{
			name:                  "should override existing file",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			expectedValidateError: nil,
			expectedAnalysisError: nil,
			outFile:               outFileTxt,
			removeOutputPath:      true,
		},
		{
			name:                  "output should be written to default txt output file",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			expectedValidateError: nil,
			expectedAnalysisError: nil,
			outputToFile:          true,
			outputFormat:          defaultOutputFormat,
		},
		{
			name:                  "output should be written to default md output file",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			expectedValidateError: nil,
			expectedAnalysisError: nil,
			outputToFile:          true,
			outputFormat:          "md",
		},
		{
			name:                  "output should be written to default csv output file",
			inputFolderPath1:      "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			expectedValidateError: nil,
			expectedAnalysisError: nil,
			outputToFile:          true,
			outputFormat:          "csv",
		},
		{
			name:                  "acs example output should be written to default txt output file",
			inputFolderPath1:      "testdata/acs-security-demos",
			inputFolderPath2:      "testdata/acs-security-demos-new-version",
			expectedValidateError: nil,
			expectedAnalysisError: nil,
			outputToFile:          true,
			outputFormat:          "txt",
		},
		{
			name:                  "acs example output should be written to default md output file",
			inputFolderPath1:      "testdata/acs-security-demos",
			inputFolderPath2:      "testdata/acs-security-demos-new-version",
			expectedValidateError: nil,
			expectedAnalysisError: nil,
			outputToFile:          true,
			outputFormat:          "md",
		},
		{
			name:                  "acs example output should be written to default csv output file",
			inputFolderPath1:      "testdata/acs-security-demos",
			inputFolderPath2:      "testdata/acs-security-demos-new-version",
			expectedValidateError: nil,
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
				outputFilePath:        tt.outFile,
				removeOutputPath:      tt.removeOutputPath,
				outputToFile:          tt.outputToFile,
				outputFormat:          tt.outputFormat,
				env:                   env,
			}

			analyzer, err := diffNetpolCmd.construct([]string{})
			d.Assert().NoError(err)

			err = diffNetpolCmd.validate()
			if tt.expectedValidateError != nil {
				d.Require().Error(err)
				d.Assert().ErrorIs(err, tt.expectedValidateError)
				return
			}
			d.Assert().NoError(err)

			err = diffNetpolCmd.analyzeConnectivityDiff(analyzer)
			if tt.expectedAnalysisError != nil {
				d.Require().Error(err)
				if tt.errStringContainment {
					d.Assert().Contains(err.Error(), tt.expectedAnalysisError.Error())
				} else {
					d.Assert().ErrorIs(err, tt.expectedAnalysisError)
				}
			} else {
				d.Assert().NoError(err)
			}

			if tt.outFile != "" && tt.expectedAnalysisError == nil && tt.expectedValidateError == nil {
				_, err := os.Stat(tt.outFile)
				d.Assert().NoError(err) // out file should exist
			}

			if tt.outputToFile && tt.outFile == "" && tt.expectedAnalysisError == nil && tt.expectedValidateError == nil {
				defaultFile := diffNetpolCmd.getDefaultFileName()
				formatSuffix := ""
				if tt.outputFormat != "" {
					d.Assert().Contains(defaultFile, tt.outputFormat)
					formatSuffix = tt.outputFormat
				} else {
					formatSuffix = defaultOutputFormat
				}
				output, err := os.ReadFile(defaultFile)
				d.Assert().NoError(err)

				expectedOutput, err := os.ReadFile(path.Join(tt.inputFolderPath2, "diff_output."+formatSuffix))
				d.Assert().NoError(err)
				d.Equal(string(expectedOutput), string(output))

				d.Assert().NoError(os.Remove(defaultFile))
			}
		})
	}

}
