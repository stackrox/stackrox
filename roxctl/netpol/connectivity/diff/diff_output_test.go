package diff

import (
	"os"
	"path"
	"strings"

	"github.com/stackrox/rox/roxctl/common/environment/mocks"
)

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
	cases := map[string]struct {
		inputFolderPath1 string
		inputFolderPath2 string
		outFile          string
		removeOutputPath bool
		outputFormat     string
		expectedErrors   []string
	}{
		"Not supported output format should result an error in formatting connectivity diff": {
			inputFolderPath1: "testdata/netpol-analysis-example-minimal",
			inputFolderPath2: "testdata/netpol-diff-example-minimal",
			outFile:          "",
			outputFormat:     "docx",
			expectedErrors:   []string{"formatting connectivity diff: docx output format is not supported"},
		},
		"Existing output file input with using remove flag should override existing output file": {
			inputFolderPath1: "testdata/netpol-analysis-example-minimal",
			inputFolderPath2: "testdata/netpol-diff-example-minimal",
			outFile:          outFileName, // existing file
			outputFormat:     "",
			removeOutputPath: true,
			expectedErrors:   []string{},
		},
		"Testing Diff between two dirs output should be written to default txt output file": {
			inputFolderPath1: "testdata/netpol-analysis-example-minimal",
			inputFolderPath2: "testdata/netpol-diff-example-minimal",
			outFile:          "",
			outputFormat:     defaultOutputFormat,
			expectedErrors:   []string{},
		},
		"Testing Diff between two dirs output should be written to default md output file": {
			inputFolderPath1: "testdata/netpol-analysis-example-minimal",
			inputFolderPath2: "testdata/netpol-diff-example-minimal",
			outFile:          "",
			outputFormat:     "md",
			expectedErrors:   []string{},
		},
		"Testing Diff between two dirs output should be written to default csv output file": {
			inputFolderPath1: "testdata/netpol-analysis-example-minimal",
			inputFolderPath2: "testdata/netpol-diff-example-minimal",
			outFile:          "",
			outputFormat:     "csv",
			expectedErrors:   []string{},
		},
		"ACS example output should be written to default txt output file": {
			inputFolderPath1: "testdata/acs-security-demos",
			inputFolderPath2: "testdata/acs-security-demos-new-version",
			outFile:          "",
			outputFormat:     "txt",
			expectedErrors:   []string{},
		},
		"ACS example output should be written to default md output file": {
			inputFolderPath1: "testdata/acs-security-demos",
			inputFolderPath2: "testdata/acs-security-demos-new-version",
			outFile:          "",
			outputFormat:     "md",
			expectedErrors:   []string{},
		},
		"ACS example output should be written to default csv output file": {
			inputFolderPath1: "testdata/acs-security-demos",
			inputFolderPath2: "testdata/acs-security-demos-new-version",
			outFile:          "",
			outputFormat:     "csv",
			expectedErrors:   []string{},
		},
	}

	for name, tt := range cases {
		d.Run(name, func() {
			env, _, _ := mocks.NewEnvWithConn(nil, d.T())
			diffNetpolCmd := diffNetpolCommand{
				inputFolderPath1:      tt.inputFolderPath1,
				inputFolderPath2:      tt.inputFolderPath2,
				outputFilePath:        tt.outFile,
				removeOutputPath:      tt.removeOutputPath,
				outputToFile:          true,
				outputFormat:          tt.outputFormat,
				env:                   env,
				stopOnFirstError:      false,
				treatWarningsAsErrors: false,
			}

			analyzer, err := diffNetpolCmd.construct()
			d.NoError(err)

			err = diffNetpolCmd.validate()
			d.NoError(err)

			// Skip asserting on warnings and errors as those are tested in diff_test.go
			_, errs := diffNetpolCmd.analyzeConnectivityDiff(analyzer)
			if len(errs) > 0 {
				return
			}

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
