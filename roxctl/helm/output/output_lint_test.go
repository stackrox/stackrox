package output

import (
	"fmt"
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stackrox/rox/roxctl/helm/internal/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"helm.sh/helm/v3/pkg/lint"
	"helm.sh/helm/v3/pkg/lint/support"
)

const (
	maxTolerableSev = support.WarningSev
)

var (
	lintNamespaces = []string{"default", "stackrox"}
)

type HelmLintTestSuite struct {
	suite.Suite
}

func (suite *HelmLintTestSuite) SetupTest() {
	testutils.SetVersion(suite.T(), version.Versions{
		MainVersion:      "3.0.55.0",
		ScannerVersion:   "3.0.55.0",
		CollectorVersion: "3.0.55.0",
		GitCommit:        "face2face",
	})
	testbuildinfo.SetForTest(suite.T())
}

func (suite *HelmLintTestSuite) TestHelmOutput() {
	tests := []struct {
		imageFlavor string
		rhacsFlag   bool
		wantErr     bool
	}{
		// rhacs = true
		{"", true, true},                       // error, --rhacs and --images-default!=rhacs returns conflict
		{imageDefaultsRHACS, true, false},      // no error, --images-default=rhacs and --rhacs provided
		{imageDefaultsDevelopment, true, true}, // error, --rhacs and --images-default!=rhacs returns conflict
		{imageDefaultsStackrox, true, true},    // error, --rhacs and --images-default!=rhacs returns conflict
		// rhacs = false
		{"", false, true},                        // error, invalid value of --images-default
		{"dummy", false, true},                   // error, invalid value of --images-default
		{imageDefaultsRHACS, false, false},       // no error, valid value of --images-default
		{imageDefaultsDevelopment, false, false}, // no error, valid value of --images-default
		{imageDefaultsStackrox, false, false},    // no error, valid value of --images-default
	}

	for _, tt := range tests {
		tt := tt
		for chartName := range common.ChartTemplates {
			suite.T().Run(fmt.Sprintf("%s-rhacs-%t-image-defaults-%s", chartName, tt.rhacsFlag, tt.imageFlavor), func(t *testing.T) {
				outputDir, err := os.MkdirTemp("", "roxctl-helm-output-lint-")
				require.NoError(suite.T(), err)
				err = outputHelmChart(chartName, outputDir, true, tt.rhacsFlag, tt.imageFlavor, false, "")
				defer func() {
					_ = os.RemoveAll(outputDir)
				}()
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	}
}

func (suite *HelmLintTestSuite) TestHelmLint() {
	for chartName := range common.ChartTemplates {
		for _, imageFlavor := range []string{imageDefaultsDevelopment, imageDefaultsStackrox, imageDefaultsRHACS} {
			suite.T().Run(fmt.Sprintf("%s-imageFlavor-%s", chartName, imageFlavor), func(t *testing.T) {
				testChartLint(t, chartName, false, imageFlavor)
			})
		}
		for _, rhacs := range []bool{false, true} {
			suite.T().Run(fmt.Sprintf("%s-rhacs-%t", chartName, rhacs), func(t *testing.T) {
				testChartLint(t, chartName, rhacs, "rhacs") // rhacs is default value for imageFlavor
			})
		}
	}
}

func TestHelmLint(t *testing.T) {
	suite.Run(t, new(HelmLintTestSuite))
}

func testChartLint(t *testing.T, chartName string, rhacs bool, imageFlavor string) {
	const noDebug = false
	const noDebugChartPath = ""
	outputDir, err := os.MkdirTemp("", "roxctl-helm-output-lint-")
	require.NoError(t, err)

	defer func() {
		_ = os.RemoveAll(outputDir)
	}()

	err = outputHelmChart(chartName, outputDir, true, rhacs, imageFlavor, noDebug, noDebugChartPath)
	require.NoErrorf(t, err, "failed to output helm chart %s", chartName)

	for _, ns := range lintNamespaces {
		t.Run(fmt.Sprintf("namespace=%s", ns), func(t *testing.T) {
			testChartInNamespaceLint(t, outputDir, ns)
		})
	}
}

func testChartInNamespaceLint(t *testing.T, chartDir string, namespace string) {
	linter := lint.All(chartDir, nil, namespace, false)

	assert.LessOrEqualf(t, linter.HighestSeverity, maxTolerableSev, "linting chart produced warnings with severity %v", linter.HighestSeverity)
	for _, msg := range linter.Messages {
		fmt.Fprintln(os.Stderr, msg.Error())
		assert.LessOrEqual(t, msg.Severity, maxTolerableSev, msg.Error())
	}
}
