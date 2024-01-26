package diff

import (
	goerrors "errors"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
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

func (d *diffAnalyzeNetpolTestSuite) TestDiffAnalyzerBehaviour() {
	cases := map[string]struct {
		name                     string
		inputFolderPath1         string
		inputFolderPath2         string
		strict                   bool
		stopOnFirstErr           bool
		expectedAnalyzerErrors   []string
		expectedAnalyzerWarnings []string
	}{
		"Testing Diff between two empty dirs should return analysis error": {
			inputFolderPath1: "testdata/empty-yamls",
			inputFolderPath2: "testdata/empty-yamls",
			expectedAnalyzerErrors: []string{
				"at dir1: no relevant Kubernetes workload resources found",
				"at dir2: no relevant Kubernetes workload resources found",
			},
			expectedAnalyzerWarnings: []string{
				"at dir1: no relevant Kubernetes network policy resources found",
				"at dir2: no relevant Kubernetes network policy resources found",
				"unable to decode \"testdata/empty-yamls/empty.yaml\"",
				"unable to decode \"testdata/empty-yamls/empty.yaml\"",
				"unable to decode \"testdata/empty-yamls/empty2.yaml\"",
				"unable to decode \"testdata/empty-yamls/empty2.yaml\"",
			},
		},
		"Testing Diff between two dirs should run successfully without errors": {
			inputFolderPath1:         "testdata/netpol-analysis-example-minimal",
			inputFolderPath2:         "testdata/netpol-diff-example-minimal",
			expectedAnalyzerErrors:   []string{},
			expectedAnalyzerWarnings: []string{},
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

			warns, errs := diffNetpolCmd.analyzeConnectivityDiff(analyzer)
			d.Require().Lenf(errs, len(tt.expectedAnalyzerErrors), "number errors should be %d", len(tt.expectedAnalyzerErrors))
			d.Require().Lenf(warns, len(tt.expectedAnalyzerWarnings), "number warnings should be %d", len(tt.expectedAnalyzerWarnings))

			for _, expError := range tt.expectedAnalyzerErrors {
				if expError != "" {
					d.Require().Error(goerrors.Join(errs...))
					d.Assert().ErrorContains(goerrors.Join(errs...), expError)
				} else {
					d.Assert().NoError(goerrors.Join(errs...))
				}
			}
			for _, expWarn := range tt.expectedAnalyzerWarnings {
				if expWarn != "" {
					d.Require().Error(goerrors.Join(warns...))
					d.Assert().ErrorContains(goerrors.Join(warns...), expWarn)
				} else {
					d.Assert().NoError(goerrors.Join(warns...))
				}
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
		name             string
		inputFolderPath1 string
		inputFolderPath2 string
		outFile          string
		removeOutputPath bool
		outputFormat     string
		expectedError    string
	}{
		{
			name:             "Not supported output format should result an error in formatting connectivity diff",
			inputFolderPath1: "testdata/netpol-analysis-example-minimal",
			inputFolderPath2: "testdata/netpol-diff-example-minimal",
			outFile:          "",
			outputFormat:     "docx",
			expectedError:    "docx output format is not supported.",
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

			_, errs := diffNetpolCmd.analyzeConnectivityDiff(analyzer)
			if tt.expectedError != "" {
				d.Require().Error(goerrors.Join(errs...))
				d.Assert().ErrorContains(goerrors.Join(errs...), tt.expectedError)
				return // we got an error, so there is no need to test the diff result
			}

			d.Assert().NoError(goerrors.Join(errs...))

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
