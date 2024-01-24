package connectivitymap

import (
	"os"
	"path"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/common/npg"
	"github.com/stretchr/testify/suite"
)

func TestAnalyzeNetpolCommand(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(analyzeNetpolTestSuite))
}

type analyzeNetpolTestSuite struct {
	suite.Suite
}

func (d *analyzeNetpolTestSuite) TestAnalyzeNetpol() {
	tmpOutFileName := d.T().TempDir() + "/out"
	outFileTxt := tmpOutFileName + ".txt"
	outFileJSON := tmpOutFileName + ".json"
	outFileMD := tmpOutFileName + ".md"
	outFileCSV := tmpOutFileName + ".csv"
	outFileDOT := tmpOutFileName + ".dot"
	cases := []struct {
		name                  string
		inputFolderPath       string
		expectedAnalysisError string
		expectedValidateError error
		treatWarningsAsErrors bool
		stopOnFirstErr        bool
		outFile               string
		outputToFile          bool
		focusWorkload         string
		outputFormat          string
		removeOutputPath      bool
	}{
		{
			name:                  "Not existing inputFolderPath should print error about path not existing but attempt analysis",
			inputFolderPath:       "/tmp/xxx",
			stopOnFirstErr:        false,
			treatWarningsAsErrors: false,
			expectedAnalysisError: "there were errors",
		},
		{
			name:                  "Not existing inputFolderPath should stop on first error about path not existing",
			inputFolderPath:       "/tmp/xxx",
			stopOnFirstErr:        true,
			treatWarningsAsErrors: false,
			expectedAnalysisError: "does not exist",
		},
		{
			name:                  "happyPath",
			inputFolderPath:       "testdata/minimal",
			expectedAnalysisError: "",
		},
		{
			name:                  "errors with no resources found",
			inputFolderPath:       "testdata/empty-yamls",
			expectedAnalysisError: npg.ErrErrors.Error(),
		},
		{
			name:                  "treating warnings as errors",
			inputFolderPath:       "testdata/minimal-with-invalid-doc",
			expectedAnalysisError: npg.ErrWarnings.Error(),
			treatWarningsAsErrors: true,
		},
		{
			name:                  "warnings not indicated without strict",
			inputFolderPath:       "testdata/minimal-with-invalid-doc",
			expectedAnalysisError: "",
		},
		{
			name:                  "stopOnFistError",
			inputFolderPath:       "testdata/dirty", // yaml document malformed
			expectedAnalysisError: npg.ErrErrors.Error(),
			stopOnFirstErr:        true,
		},
		{
			name:                  "output should be written to a single file",
			inputFolderPath:       "testdata/minimal",
			expectedValidateError: nil,
			expectedAnalysisError: "",
			outFile:               outFileTxt,
			removeOutputPath:      false,
		},
		{
			name:                  "should return error that the file already exists",
			inputFolderPath:       "testdata/minimal",
			expectedValidateError: errox.AlreadyExists,
			expectedAnalysisError: "",
			outFile:               outFileTxt,
			removeOutputPath:      false,
		},
		{
			name:                  "should override existing file",
			inputFolderPath:       "testdata/minimal",
			expectedValidateError: nil,
			expectedAnalysisError: "",
			outFile:               outFileTxt,
			removeOutputPath:      true,
		},
		{
			name:                  "output should be written to default txt output file",
			inputFolderPath:       "testdata/minimal",
			expectedValidateError: nil,
			expectedAnalysisError: "",
			outputToFile:          true,
			outputFormat:          defaultOutputFormat,
		},
		{
			name:                  "output should be focused to a workload",
			inputFolderPath:       "testdata/minimal",
			focusWorkload:         "default/backend",
			expectedValidateError: nil,
			expectedAnalysisError: "",
		},
		{
			name:                  "not supported output format",
			inputFolderPath:       "testdata/minimal",
			outputFormat:          "docx",
			expectedAnalysisError: "docx output format is not supported.",
		},
		{
			name:                  "generate output in json format",
			inputFolderPath:       "testdata/minimal",
			outputFormat:          "json",
			expectedAnalysisError: "",
			expectedValidateError: nil,
			outFile:               outFileJSON,
		},
		{
			name:                  "generate output in md format",
			inputFolderPath:       "testdata/minimal",
			outputFormat:          "md",
			expectedAnalysisError: "",
			expectedValidateError: nil,
			outFile:               outFileMD,
		},
		{
			name:                  "generate output in csv format",
			inputFolderPath:       "testdata/minimal",
			outputFormat:          "csv",
			expectedAnalysisError: "",
			expectedValidateError: nil,
			outFile:               outFileCSV,
		},
		{
			name:                  "generate output in dot format",
			inputFolderPath:       "testdata/minimal",
			outputFormat:          "dot",
			expectedAnalysisError: "",
			expectedValidateError: nil,
			outFile:               outFileDOT,
		},
		{
			name:                  "openshift resources are recognized by the serializer with k8s resources",
			inputFolderPath:       "testdata/frontend-security",
			expectedAnalysisError: "",
			expectedValidateError: nil,
		},
		{
			name:                  "output should be written to default json output file",
			inputFolderPath:       "testdata/minimal",
			expectedValidateError: nil,
			expectedAnalysisError: "",
			outputToFile:          true,
			outputFormat:          "json",
		},
		{
			name:                  "generate connections list with ingress controller",
			inputFolderPath:       "testdata/acs-security-demos",
			expectedValidateError: nil,
			expectedAnalysisError: "",
			outputToFile:          true,
		},
	}

	for _, tt := range cases {
		tt := tt
		d.Run(tt.name, func() {
			env, _, _ := mocks.NewEnvWithConn(nil, d.T())
			analyzeNetpolCmd := Cmd{
				stopOnFirstError:      tt.stopOnFirstErr,
				treatWarningsAsErrors: tt.treatWarningsAsErrors,
				inputFolderPath:       "", // set through construct
				outputFilePath:        tt.outFile,
				removeOutputPath:      tt.removeOutputPath,
				outputToFile:          tt.outputToFile,
				focusWorkload:         tt.focusWorkload,
				outputFormat:          tt.outputFormat,
				env:                   env,
			}

			analyzer, err := analyzeNetpolCmd.construct([]string{tt.inputFolderPath})
			d.Assert().NoError(err)

			err = analyzeNetpolCmd.validate()
			if tt.expectedValidateError != nil {
				d.Require().Error(err)
				d.Assert().ErrorIs(err, tt.expectedValidateError)
				return
			}
			d.Assert().NoError(err)

			err = analyzeNetpolCmd.analyzeNetpols(analyzer)
			if tt.expectedAnalysisError != "" {
				d.Require().Error(err)
				d.Assert().Contains(err.Error(), tt.expectedAnalysisError)
			} else {
				d.Assert().NoError(err)
			}

			if tt.outFile != "" && tt.expectedAnalysisError == "" && tt.expectedValidateError == nil {
				_, err := os.Stat(tt.outFile)
				d.Assert().NoError(err) // out file should exist
			}

			if tt.outputToFile && tt.outFile == "" && tt.expectedAnalysisError == "" && tt.expectedValidateError == nil {
				defaultFile := analyzeNetpolCmd.getDefaultFileName()
				formatSuffix := ""
				if tt.outputFormat != "" {
					d.Assert().Contains(defaultFile, tt.outputFormat)
					formatSuffix = tt.outputFormat
				} else {
					formatSuffix = defaultOutputFormat
				}
				output, err := os.ReadFile(defaultFile)
				d.Assert().NoError(err)

				expectedOutput, err := os.ReadFile(path.Join(tt.inputFolderPath, "output."+formatSuffix))
				d.Assert().NoError(err)
				d.Equal(string(expectedOutput), string(output))

				d.Assert().NoError(os.Remove(defaultFile))
			}
		})
	}

}
