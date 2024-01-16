package diff

import (
	goerrors "errors"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/pkg/errors"
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
		tt := tt
		d.Run(tt.name, func() {
			env, _, _ := mocks.NewEnvWithConn(nil, d.T())
			diffNetpolCmd := diffNetpolCommand{
				inputFolderPath1: tt.inputFolderPath1,
				inputFolderPath2: tt.inputFolderPath2,
				outputFilePath:   tt.outFile,
				env:              env,
			}

			_, err := diffNetpolCmd.construct()
			d.NoError(err)

			err = diffNetpolCmd.validate()
			if tt.expectedValidateError != nil {
				d.Require().Error(err)
				d.ErrorIs(err, tt.expectedValidateError)
				return
			}
			d.NoError(err)
		})
	}
}

func (d *diffAnalyzeNetpolTestSuite) TestProcessInput() {
	cases := map[string]struct {
		inputFolderPath1 string
		inputFolderPath2 string
		strict           bool
		stopOnFirstErr   bool
		expectedWarn     error
		expectedErr      error
	}{
		"Not existing input folder paths should result in error 'errox.NotFound'": {
			inputFolderPath1: "/tmp/xxx",
			inputFolderPath2: "/tmp/xxx",
			expectedWarn:     nil,
			expectedErr:      errox.NotFound,
		},
		"Inputs with no resources should result in general NP-Guard error": {
			inputFolderPath1: "testdata/empty-yamls",
			inputFolderPath2: "testdata/empty-yamls",
			strict:           false,
			expectedWarn:     npg.ErrWarnings,
			expectedErr:      nil,
		},
		"Inputs with no resources should result in general NP-Guard error when run with --fail": {
			inputFolderPath1: "testdata/empty-yamls",
			inputFolderPath2: "testdata/empty-yamls",
			strict:           true,
			expectedWarn:     nil,
			expectedErr:      npg.ErrWarnings,
		},
		"Treating warnings as errors should result in error of type 'npg.ErrWarnings'": {
			inputFolderPath1: "testdata/acs-zeroday-with-invalid-doc",
			inputFolderPath2: "testdata/acs-zeroday-with-invalid-doc",
			strict:           true,
			expectedWarn:     nil,
			expectedErr:      npg.ErrWarnings,
		},
		"Warnings on invalid input docs without using strict flag should not be treated as errors": {
			inputFolderPath1: "testdata/acs-zeroday-with-invalid-doc",
			inputFolderPath2: "testdata/acs-zeroday-with-invalid-doc",
			strict:           false,
			expectedWarn:     npg.ErrWarnings,
			expectedErr:      nil,
		},
		"Stop on first error with malformed yaml inputs should stop with general NP-Guard error as warning": {
			inputFolderPath1: "testdata/dirty", // yaml document malformed
			inputFolderPath2: "testdata/dirty",
			stopOnFirstErr:   true,
			strict:           false,
			expectedWarn:     npg.ErrWarnings,
			expectedErr:      nil,
		},
		"Stop on first error with malformed yaml inputs should stop with general NP-Guard error as error": {
			inputFolderPath1: "testdata/dirty", // yaml document malformed
			inputFolderPath2: "testdata/dirty",
			stopOnFirstErr:   true,
			strict:           true,
			expectedWarn:     nil,
			expectedErr:      npg.ErrWarnings,
		},
		"Testing Diff between two dirs should run successfully without errors": {
			inputFolderPath1: "testdata/netpol-analysis-example-minimal",
			inputFolderPath2: "testdata/netpol-diff-example-minimal",
			expectedWarn:     nil,
			expectedErr:      nil,
		},
	}

	for name, tt := range cases {
		tt := tt
		d.Run(name, func() {
			env, _, _ := mocks.NewEnvWithConn(nil, d.T())
			diffNetpolCmd := diffNetpolCommand{
				stopOnFirstError:      tt.stopOnFirstErr,
				treatWarningsAsErrors: tt.strict,
				inputFolderPath1:      tt.inputFolderPath1,
				inputFolderPath2:      tt.inputFolderPath2,
				env:                   env,
			}

			d.NoError(diffNetpolCmd.validate())

			_, _, warn, err := diffNetpolCmd.processInput()
			if tt.expectedWarn != nil {
				d.Require().Error(warn)
				d.ErrorIs(warn, tt.expectedWarn)
			} else {
				d.NoError(warn, "Received unexpected warning")
			}
			if tt.expectedErr != nil {
				d.Require().Error(err)
				d.ErrorIs(err, tt.expectedErr)
			} else {
				d.NoError(err)
			}
		})
	}
}

func (d *diffAnalyzeNetpolTestSuite) TestDiffAnalyzerBehaviour() {
	cases := map[string]struct {
		name                   string
		inputFolderPath1       string
		inputFolderPath2       string
		strict                 bool
		stopOnFirstErr         bool
		expectedAnalyzerErrors []error
		expectedError          error
	}{
		"Testing Diff between two empty dirs should return analysis error": {
			inputFolderPath1: "testdata/empty-yamls",
			inputFolderPath2: "testdata/empty-yamls",
			expectedAnalyzerErrors: []error{
				errors.New("at dir1: no relevant Kubernetes network policy resources found"),
				errors.New("at dir2: no relevant Kubernetes network policy resources found"),
				errors.New("at dir1: no relevant Kubernetes workload resources found"),
				errors.New("at dir2: no relevant Kubernetes workload resources found"),
			},
			expectedError: npg.ErrErrors,
		},
		"Testing Diff between two dirs should run successfully without errors": {
			inputFolderPath1:       "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:       "testdata/netpol-diff-example-minimal",
			expectedAnalyzerErrors: []error{},
			expectedError:          nil,
		},
	}

	for name, tt := range cases {
		tt := tt
		d.Run(name, func() {
			env, _, _ := mocks.NewEnvWithConn(nil, d.T())
			diffNetpolCmd := diffNetpolCommand{
				stopOnFirstError:      tt.stopOnFirstErr,
				treatWarningsAsErrors: tt.strict,
				inputFolderPath1:      tt.inputFolderPath1,
				inputFolderPath2:      tt.inputFolderPath2,
				env:                   env,
			}

			analyzer, err := diffNetpolCmd.construct()
			d.NoError(err)

			err = diffNetpolCmd.validate()
			d.NoError(err)

			err = diffNetpolCmd.analyzeConnectivityDiff(analyzer)
			if tt.expectedError != nil {
				d.Require().Error(err)
				d.ErrorIs(err, tt.expectedError)
			} else {
				d.NoError(err, "expected no error, but got: %v", err)
			}

			// convert custom errors to common errors
			analyzerErrors := make([]error, len(analyzer.Errors()))
			for i, diffError := range analyzer.Errors() {
				analyzerErrors[i] = diffError.Error()
			}

			for _, err2 := range tt.expectedAnalyzerErrors {
				d.ErrorContains(goerrors.Join(analyzerErrors...), err2.Error())
			}

			if tt.expectedAnalyzerErrors == nil || len(tt.expectedAnalyzerErrors) == 0 {
				d.NoError(err, "expected no analyser error, but got: %v", goerrors.Join(analyzerErrors...))
			}

		})
	}
}

func (d *diffAnalyzeNetpolTestSuite) createOutFile() string {
	tempDir := d.T().TempDir()
	tmpOutFile, err := os.CreateTemp(tempDir, "out")
	d.NoError(err)
	return tmpOutFile.Name()
}

func (d *diffAnalyzeNetpolTestSuite) assertFileContentsMatch(expectedString, fileName string) {
	d.FileExists(fileName)
	fileContents, err := os.ReadFile(fileName)
	d.NoError(err)
	// Ignore the last trailing newline
	d.Equal(strings.TrimSuffix(expectedString, "\n"), strings.TrimSuffix(string(fileContents), "\n"))
}

func (d *diffAnalyzeNetpolTestSuite) TestDiffOutput() {
	outFileName := d.createOutFile()
	cases := []struct {
		name                           string
		inputFolderPath1               string
		inputFolderPath2               string
		outFile                        string
		removeOutputPath               bool
		outputFormat                   string
		expectedWrongFormatErrContains string
	}{
		{
			name:                           "Not supported output format should result an error in formatting connectivity diff",
			inputFolderPath1:               "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:               "testdata/netpol-diff-example-minimal",
			outFile:                        "",
			outputFormat:                   "docx",
			expectedWrongFormatErrContains: "docx output format is not supported.",
		},
		{
			name:             "Existing output file input with using remove flag should override existing output file",
			inputFolderPath1: "testdata/netpol-analysis-example-minimal",
			inputFolderPath2: "testdata/netpol-diff-example-minimal",
			outFile:          outFileName, // existing file
			outputFormat:     "",
			removeOutputPath: true,
		},
		{
			name:             "Testing Diff between two dirs output should be written to default txt output file",
			inputFolderPath1: "testdata/netpol-analysis-example-minimal",
			inputFolderPath2: "testdata/netpol-diff-example-minimal",
			outFile:          "",
			outputFormat:     defaultOutputFormat,
		},
		{
			name:             "Testing Diff between two dirs output should be written to default md output file",
			inputFolderPath1: "testdata/netpol-analysis-example-minimal",
			inputFolderPath2: "testdata/netpol-diff-example-minimal",
			outFile:          "",
			outputFormat:     "md",
		},
		{
			name:             "Testing Diff between two dirs output should be written to default csv output file",
			inputFolderPath1: "testdata/netpol-analysis-example-minimal",
			inputFolderPath2: "testdata/netpol-diff-example-minimal",
			outFile:          "",
			outputFormat:     "csv",
		},
		{
			name:             "ACS example output should be written to default txt output file",
			inputFolderPath1: "testdata/acs-security-demos",
			inputFolderPath2: "testdata/acs-security-demos-new-version",
			outFile:          "",
			outputFormat:     "txt",
		},
		{
			name:             "ACS example output should be written to default md output file",
			inputFolderPath1: "testdata/acs-security-demos",
			inputFolderPath2: "testdata/acs-security-demos-new-version",
			outFile:          "",
			outputFormat:     "md",
		},
		{
			name:             "ACS example output should be written to default csv output file",
			inputFolderPath1: "testdata/acs-security-demos",
			inputFolderPath2: "testdata/acs-security-demos-new-version",
			outFile:          "",
			outputFormat:     "csv",
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
				outputToFile:     true,
				outputFormat:     tt.outputFormat,
				env:              env,
			}

			analyzer, err := diffNetpolCmd.construct()
			d.NoError(err)

			err = diffNetpolCmd.validate()
			d.NoError(err)

			err = diffNetpolCmd.analyzeConnectivityDiff(analyzer)
			if tt.expectedWrongFormatErrContains != "" {
				d.Require().Error(err)
				d.ErrorContains(err, tt.expectedWrongFormatErrContains)
				return
			}
			d.NoError(err)

			formatSuffix := tt.outputFormat
			if formatSuffix == "" {
				formatSuffix = defaultOutputFormat
			}
			outFileName := tt.outFile
			if outFileName == "" {
				outFileName = diffNetpolCmd.getDefaultFileName()
				d.Contains(outFileName, formatSuffix)
			}

			expectedOutput, err := os.ReadFile(path.Join(tt.inputFolderPath2, "diff_output."+formatSuffix))
			d.NoError(err)
			d.assertFileContentsMatch(string(expectedOutput), outFileName)
			d.NoError(os.Remove(outFileName))
		})
	}
}
