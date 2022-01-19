package output

import (
	"fmt"
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/printer"
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

type HelmChartTestSuite struct {
	suite.Suite
}

func (s *HelmChartTestSuite) SetupTest() {
	testutils.SetExampleVersion(s.T())
	testbuildinfo.SetForTest(s.T())
}

func TestHelmLint(t *testing.T) {
	suite.Run(t, new(HelmChartTestSuite))
}

func (s *HelmChartTestSuite) TestHelmOutput() {
	type testCase struct {
		flavor         string
		flavorProvided bool
		rhacs          bool
		wantErr        bool
	}
	tests := []testCase{
		{flavor: "", flavorProvided: false, rhacs: true},                                // '--rhacs' but no '--image-defaults'
		{flavor: "", flavorProvided: true, rhacs: true, wantErr: true},                  // error: collision of --rhacs and --image-defults
		{flavor: "dummy", rhacs: true, wantErr: true},                                   // any flavor (even invalid) has priority over --rhacs
		{flavor: defaults.ImageFlavorNameStackRoxIORelease, rhacs: true, wantErr: true}, // error: collision of --rhacs and --image-defults
		{flavor: "", flavorProvided: false, rhacs: false},                               // no '--rhacs' and no '--image-defaults'
		{flavor: "", flavorProvided: true, rhacs: false, wantErr: true},                 // error: collision of --rhacs and --image-defults
		{flavor: "dummy", rhacs: false, wantErr: true},
		{flavor: defaults.ImageFlavorNameStackRoxIORelease, rhacs: false},
		{flavor: defaults.ImageFlavorNameRHACSRelease, rhacs: false},
		{flavor: defaults.ImageFlavorNameRHACSRelease, rhacs: true, wantErr: true},
	}
	// development flavor can be used only on non-released builds
	if !buildinfo.ReleaseBuild {
		tests = append(tests,
			testCase{flavor: defaults.ImageFlavorNameDevelopmentBuild, rhacs: true, wantErr: true}, // error: collision of --rhacs and non-empty --image-defults
			testCase{flavor: defaults.ImageFlavorNameDevelopmentBuild, rhacs: false},
		)
	}
	testIO, _, _, _ := environment.TestIO()
	env := environment.NewCLIEnvironment(testIO, printer.DefaultColorPrinter())

	for _, tt := range tests {
		tt := tt
		for chartName := range common.ChartTemplates {
			s.Run(fmt.Sprintf("%s-rhacs-%t-image-defaults-%s", chartName, tt.rhacs, tt.flavor), func() {
				outputDir, err := os.MkdirTemp("", "roxctl-helm-output-lint-")
				s.T().Cleanup(func() {
					_ = os.RemoveAll(outputDir)
				})
				require.NoError(s.T(), err)
				if tt.flavor != "" {
					tt.flavorProvided = true
				}
				metaVals, metaValsErr := getMetaValues(tt.flavor, tt.flavorProvided, tt.rhacs, buildinfo.ReleaseBuild, env)
				err = outputHelmChart(chartName, outputDir, true, metaVals, false, "")
				if tt.wantErr {
					assert.Error(s.T(), metaValsErr)
					assert.Error(s.T(), err)
				} else {
					assert.NoError(s.T(), metaValsErr)
					assert.NoError(s.T(), err)
				}
			})
		}
	}
}

func (s *HelmChartTestSuite) TestHelmLint() {
	flavorsToTest := []string{defaults.ImageFlavorNameStackRoxIORelease, defaults.ImageFlavorNameRHACSRelease}
	if !buildinfo.ReleaseBuild {
		flavorsToTest = append(flavorsToTest, defaults.ImageFlavorNameDevelopmentBuild)
	}

	for chartName := range common.ChartTemplates {
		for _, imageFlavor := range flavorsToTest {
			s.Run(fmt.Sprintf("%s-imageFlavor-%s", chartName, imageFlavor), func() {
				testChartLint(s.T(), chartName, false, imageFlavor)
			})
		}
		for _, rhacs := range []bool{false, true} {
			s.Run(fmt.Sprintf("%s-rhacs-%t", chartName, rhacs), func() {
				testChartLint(s.T(), chartName, rhacs, defaults.ImageFlavorNameRHACSRelease)
			})
		}
	}
}

func testChartLint(t *testing.T, chartName string, rhacs bool, imageFlavor string) {
	const noDebug = false
	const noDebugChartPath = ""
	outputDir, err := os.MkdirTemp("", "roxctl-helm-output-lint-")
	t.Cleanup(func() {
		_ = os.RemoveAll(outputDir)
	})
	require.NoError(t, err)

	testIO, _, _, _ := environment.TestIO()
	env := environment.NewCLIEnvironment(testIO, printer.DefaultColorPrinter())

	if rhacs && imageFlavor != "" {
		t.Skipf("invalid parameter combination for this test")
	}
	metaVals, metaValsErr := getMetaValues(imageFlavor, imageFlavor != "", rhacs, buildinfo.ReleaseBuild, env)
	require.NoErrorf(t, metaValsErr, "failed to getMetaValues for chart %s", chartName)
	err = outputHelmChart(chartName, outputDir, true, metaVals, noDebug, noDebugChartPath)
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
