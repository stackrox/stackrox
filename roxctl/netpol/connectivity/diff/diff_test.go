package diff

import (
	"testing"

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

func (d *diffAnalyzeNetpolTestSuite) TestAnalyzeConnectivityDiffWarningsErrors() {
	cases := map[string]struct {
		inputFolderPath1      string
		inputFolderPath2      string
		stopOnFirstError      bool
		treatWarningsAsErrors bool

		expectedErrors   []string
		expectedWarnings []string
	}{
		"Errors and stopOnFirstError should cause the analysis to stop early": {
			inputFolderPath1: "testdata/dirty",
			inputFolderPath2: "testdata/netpol-diff-example-minimal",
			stopOnFirstError: true,
			expectedErrors:   []string{"at dir1: no relevant Kubernetes workload resources found"},
			expectedWarnings: []string{
				"error parsing testdata/dirty/backend.yaml",
				"error parsing testdata/dirty/frontend.yaml",
				"at dir1: no relevant Kubernetes network policy resources found",
			},
		},
		"Warnings with stopOnFirstError and treatWarningsAsErrors should stop the analysis on first warning": {
			inputFolderPath1:      "testdata/mixed",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			stopOnFirstError:      true,
			treatWarningsAsErrors: true,
			expectedErrors:        []string{},
			expectedWarnings:      []string{"error parsing testdata/mixed/dirty.yaml"},
		},
		"Warnings with treatWarningsAsErrors should not add any marker-error at this stage": {
			inputFolderPath1:      "testdata/mixed",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			stopOnFirstError:      false,
			treatWarningsAsErrors: true,
			expectedErrors:        []string{},
			expectedWarnings: []string{
				"error parsing testdata/mixed/dirty.yaml",
				"at dir1: no relevant Kubernetes network policy resources found",
			},
		},
		"Diff should be attempted despite of warnings when treatWarningsAsErrors is false": {
			inputFolderPath1:      "testdata/mixed",
			inputFolderPath2:      "testdata/netpol-diff-example-minimal",
			stopOnFirstError:      false,
			treatWarningsAsErrors: false,
			expectedErrors:        []string{},
			expectedWarnings: []string{
				"error parsing testdata/mixed/dirty.yaml",
				"at dir1: no relevant Kubernetes network policy resources found",
			},
		},
		"Testing Diff between two empty dirs should return analysis error": {
			inputFolderPath1: "testdata/empty-yamls",
			inputFolderPath2: "testdata/empty-yamls",
			expectedErrors: []string{
				"at dir1: no relevant Kubernetes workload resources found",
				"at dir2: no relevant Kubernetes workload resources found",
			},
			expectedWarnings: []string{
				"at dir1: no relevant Kubernetes network policy resources found",
				"at dir2: no relevant Kubernetes network policy resources found",
				"unable to decode \"testdata/empty-yamls/empty.yaml\"",
				"unable to decode \"testdata/empty-yamls/empty.yaml\"",
				"unable to decode \"testdata/empty-yamls/empty2.yaml\"",
				"unable to decode \"testdata/empty-yamls/empty2.yaml\"",
			},
		},
		"Testing Diff between two dirs should run successfully without errors": {
			inputFolderPath1: "testdata/netpol-analysis-example-minimal",
			inputFolderPath2: "testdata/netpol-diff-example-minimal",
			expectedErrors:   []string{},
			expectedWarnings: []string{},
		},
	}

	for name, tt := range cases {
		d.Run(name, func() {
			env, _, _ := mocks.NewEnvWithConn(nil, d.T())
			diffNetpolCmd := diffNetpolCommand{
				inputFolderPath1:      tt.inputFolderPath1,
				inputFolderPath2:      tt.inputFolderPath2,
				outputFilePath:        "/tmp/dummy",
				removeOutputPath:      true,
				outputToFile:          true,
				outputFormat:          "txt",
				env:                   env,
				stopOnFirstError:      tt.stopOnFirstError,
				treatWarningsAsErrors: tt.treatWarningsAsErrors,
			}

			analyzer, err := diffNetpolCmd.construct()
			d.NoError(err)

			err = diffNetpolCmd.validate()
			d.NoError(err)

			warns, errs := diffNetpolCmd.analyzeConnectivityDiff(analyzer)
			npg.AssertErrorsContain(d.T(), tt.expectedErrors, errs, "errors")
			npg.AssertErrorsContain(d.T(), tt.expectedWarnings, warns, "warnings")
		})
	}
}
