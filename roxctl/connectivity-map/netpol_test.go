package connectivitymap

import (
	"errors"
	"os"
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
		expectedAnalysisError error
		expectedValidateError error
		strict                bool
		stopOnFirstErr        bool
		outFile               string
		outputToFile          bool
		focusWorkload         string
		outputFormat          string
		expectedOutput        string
		removeOutputPath      bool
		errStringContainment  bool
	}{
		{
			name:                  "not existing inputFolderPath should raise 'os.ErrNotExist' error",
			inputFolderPath:       "/tmp/xxx",
			expectedAnalysisError: os.ErrNotExist,
		},
		{
			name:                  "happyPath",
			inputFolderPath:       "testdata/minimal",
			expectedAnalysisError: nil,
		},
		{
			name:                  "errors with no resources found",
			inputFolderPath:       "testdata/empty-yamls",
			expectedAnalysisError: npg.ErrErrors,
		},
		{
			name:                  "treating warnings as errors",
			inputFolderPath:       "testdata/minimal-with-invalid-doc",
			expectedAnalysisError: npg.ErrWarnings,
			strict:                true,
		},
		{
			name:                  "warnings not indicated without strict",
			inputFolderPath:       "testdata/minimal-with-invalid-doc",
			expectedAnalysisError: nil,
		},
		{
			name:                  "stopOnFistError",
			inputFolderPath:       "testdata/dirty", // yaml document malformed
			expectedAnalysisError: npg.ErrErrors,
			stopOnFirstErr:        true,
		},
		{
			name:                  "output should be written to a single file",
			inputFolderPath:       "testdata/minimal",
			expectedValidateError: nil,
			expectedAnalysisError: nil,
			outFile:               outFileTxt,
			removeOutputPath:      false,
		},
		{
			name:                  "should return error that the file already exists",
			inputFolderPath:       "testdata/minimal",
			expectedValidateError: errox.AlreadyExists,
			expectedAnalysisError: nil,
			outFile:               outFileTxt,
			removeOutputPath:      false,
		},
		{
			name:                  "should override existing file",
			inputFolderPath:       "testdata/minimal",
			expectedValidateError: nil,
			expectedAnalysisError: nil,
			outFile:               outFileTxt,
			removeOutputPath:      true,
		},
		{
			name:                  "output should be written to default txt output file",
			inputFolderPath:       "testdata/minimal",
			expectedValidateError: nil,
			expectedAnalysisError: nil,
			outputToFile:          true,
			expectedOutput: `0.0.0.0-255.255.255.255 => default/frontend[Deployment] : TCP 8080
default/backend[Deployment] => default/backend[Deployment] : All Connections
default/frontend[Deployment] => 0.0.0.0-255.255.255.255 : UDP 53
default/frontend[Deployment] => default/backend[Deployment] : TCP 9090
default/frontend[Deployment] => default/frontend[Deployment] : All Connections`,
		},
		{
			name:                  "output should be focused to a workload",
			inputFolderPath:       "testdata/minimal",
			focusWorkload:         "default/backend",
			expectedValidateError: nil,
			expectedAnalysisError: nil,
		},
		{
			name:                  "not supported output format",
			inputFolderPath:       "testdata/minimal",
			outputFormat:          "docx",
			errStringContainment:  true,
			expectedAnalysisError: errors.New("docx output format is not supported."),
		},
		{
			name:                  "generate output in json format",
			inputFolderPath:       "testdata/minimal",
			outputFormat:          "json",
			expectedAnalysisError: nil,
			expectedValidateError: nil,
			outFile:               outFileJSON,
		},
		{
			name:                  "generate output in md format",
			inputFolderPath:       "testdata/minimal",
			outputFormat:          "md",
			expectedAnalysisError: nil,
			expectedValidateError: nil,
			outFile:               outFileMD,
		},
		{
			name:                  "generate output in csv format",
			inputFolderPath:       "testdata/minimal",
			outputFormat:          "csv",
			expectedAnalysisError: nil,
			expectedValidateError: nil,
			outFile:               outFileCSV,
		},
		{
			name:                  "generate output in dot format",
			inputFolderPath:       "testdata/minimal",
			outputFormat:          "dot",
			expectedAnalysisError: nil,
			expectedValidateError: nil,
			outFile:               outFileDOT,
		},
		{
			name:                  "openshift resources are recognized by the serializer with k8s resources",
			inputFolderPath:       "testdata/frontend-security",
			expectedAnalysisError: nil,
			expectedValidateError: nil,
		},
		{
			name:                  "output should be written to default json output file",
			inputFolderPath:       "testdata/minimal",
			expectedValidateError: nil,
			expectedAnalysisError: nil,
			outputToFile:          true,
			outputFormat:          "json",
			expectedOutput: `[
  {
    "src": "0.0.0.0-255.255.255.255",
    "dst": "default/frontend[Deployment]",
    "conn": "TCP 8080"
  },
  {
    "src": "default/backend[Deployment]",
    "dst": "default/backend[Deployment]",
    "conn": "All Connections"
  },
  {
    "src": "default/frontend[Deployment]",
    "dst": "0.0.0.0-255.255.255.255",
    "conn": "UDP 53"
  },
  {
    "src": "default/frontend[Deployment]",
    "dst": "default/backend[Deployment]",
    "conn": "TCP 9090"
  },
  {
    "src": "default/frontend[Deployment]",
    "dst": "default/frontend[Deployment]",
    "conn": "All Connections"
  }
]`,
		},
	}

	for _, tt := range cases {
		tt := tt
		d.Run(tt.name, func() {
			env, _, _ := mocks.NewEnvWithConn(nil, d.T())
			analyzeNetpolCmd := analyzeNetpolCommand{
				stopOnFirstError:      tt.stopOnFirstErr,
				treatWarningsAsErrors: tt.strict,
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
				defaultFile := analyzeNetpolCmd.getDefaultFileName()
				if tt.outputFormat != "" {
					d.Assert().Contains(defaultFile, tt.outputFormat)
				}
				output, err := os.ReadFile(defaultFile)
				d.Assert().NoError(err)
				d.Equal(tt.expectedOutput, string(output))
				d.Assert().NoError(err)
				d.Assert().NoError(os.Remove(defaultFile))
			}
		})
	}

}
