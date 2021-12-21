package output

import (
	"fmt"
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo"
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

func TestHelmLint(t *testing.T) {
	suite.Run(t, new(HelmLintTestSuite))
}

func (suite *HelmLintTestSuite) TestHelmOutput() {
	type testCase struct {
		flavor  string
		rhacs   bool
		wantErr bool
	}
	tests := []testCase{
		{"", true, false}, // '--rhacs' but no '--image-defaults'
		{"dummy", true, true},
		{imageDefaultsStackrox, true, false},
		{"", false, false}, // no '--rhacs' and no '--image-defaults'
		{"dummy", false, true},
		{imageDefaultsStackrox, false, false},
	}
	// development flavor can be used only on non-released builds
	if !buildinfo.ReleaseBuild {
		tests = append(tests, []testCase{
			{imageDefaultsDevelopment, true, false},
			{imageDefaultsDevelopment, false, false},
		}...)
	}

	for _, tt := range tests {
		tt := tt
		for chartName := range common.ChartTemplates {
			suite.T().Run(fmt.Sprintf("%s-rhacs-%t-image-defaults-%s", chartName, tt.rhacs, tt.flavor), func(t *testing.T) {
				outputDir, err := os.MkdirTemp("", "roxctl-helm-output-lint-")
				require.NoError(suite.T(), err)
				err = outputHelmChart(chartName, outputDir, true, tt.rhacs, tt.flavor, false, "")
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
	flavorsToTest := []string{imageDefaultsStackrox}
	if !buildinfo.ReleaseBuild {
		flavorsToTest = append(flavorsToTest, imageDefaultsDevelopment)
	}

	for chartName := range common.ChartTemplates {
		for _, imageFlavor := range flavorsToTest {
			suite.T().Run(fmt.Sprintf("%s-imageFlavor-%s", chartName, imageFlavor), func(t *testing.T) {
				testChartLint(t, chartName, false, imageFlavor)
			})
		}
		for _, rhacs := range []bool{false, true} {
			suite.T().Run(fmt.Sprintf("%s-rhacs-%t", chartName, rhacs), func(t *testing.T) {
				testChartLint(t, chartName, rhacs, imageDefaultsDefault) // TODO(RS-380): Use RHACS as the new default
			})
		}
	}
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
