package output

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/io"
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
}

func TestHelmLint(t *testing.T) {
	suite.Run(t, new(HelmChartTestSuite))
}

// TestOutputHelmChart demonstrates current behavior of outputHelmChart.
// This test doesn't define a contract, so changes to the behavior are possible.
// The contract is tested in the roxctl e2e tests.
func (s *HelmChartTestSuite) TestOutputHelmChart() {
	type testCase struct {
		flavor         string
		flavorProvided bool // flavorProvided will be changed to true for non-empty 'flavor'
		rhacs          bool
		wantErr        bool
	}
	tests := []testCase{
		// Group: Invalid --image-defaults, no --rhacs
		// outputHelmChart currently does not guess the default flavor, valid default flavor comes from command line flag setup
		{flavor: "", flavorProvided: false, rhacs: false, wantErr: true},
		{flavor: "", flavorProvided: true, rhacs: false, wantErr: true},
		{flavor: "dummy", rhacs: false, wantErr: true},

		// Group: Valid --image-defaults, no --rhacs
		{flavor: defaults.ImageFlavorNameDevelopmentBuild, rhacs: false},
		{flavor: defaults.ImageFlavorNameStackRoxIORelease, rhacs: false},
		{flavor: defaults.ImageFlavorNameRHACSRelease, rhacs: false},
		{flavor: defaults.ImageFlavorNameOpenSource, rhacs: false},

		// Group: --rhacs only (test backwards-compatibility with versions < v3.68)
		{flavor: "", flavorProvided: false, rhacs: true},

		// Group: Both --image-defaults and --rhacs provided
		// Providing both flags shall produce flag-collision error
		{flavor: "", flavorProvided: true, rhacs: true, wantErr: true},
		{flavor: "dummy", rhacs: true, wantErr: true},
		{flavor: defaults.ImageFlavorNameDevelopmentBuild, rhacs: true, wantErr: true},
		{flavor: defaults.ImageFlavorNameStackRoxIORelease, rhacs: true, wantErr: true},
		{flavor: defaults.ImageFlavorNameRHACSRelease, rhacs: true, wantErr: true},
		{flavor: defaults.ImageFlavorNameOpenSource, rhacs: true, wantErr: true},
	}

	testIO, _, _, _ := io.TestIO()
	env := environment.NewTestCLIEnvironment(s.T(), testIO, printer.DefaultColorPrinter())

	for _, tt := range tests {
		tt := tt
		for chartName := range common.ChartTemplates {
			s.Run(fmt.Sprintf("%s-rhacs-%t-flavorProvided-%t-image-defaults-%s", chartName, tt.rhacs, tt.flavorProvided, tt.flavor), func() {
				outputDir := s.T().TempDir()
				if tt.flavor != "" {
					tt.flavorProvided = true
				}
				err := executeHelpOutputCommand(chartName, outputDir, true, tt.flavor, tt.flavorProvided, tt.rhacs, env)
				if tt.wantErr {
					assert.Error(s.T(), err)
				} else {
					assert.NoError(s.T(), err)
				}
			})
		}
	}
}

func (s *HelmChartTestSuite) TestHelmLint() {
	flavorsToTest := []string{
		defaults.ImageFlavorNameDevelopmentBuild,
		defaults.ImageFlavorNameStackRoxIORelease,
		defaults.ImageFlavorNameRHACSRelease,
		defaults.ImageFlavorNameOpenSource,
	}

	for chartName := range common.ChartTemplates {
		for _, imageFlavor := range flavorsToTest {
			s.Run(fmt.Sprintf("%s-imageFlavor-%s", chartName, imageFlavor), func() {
				testChartLint(s.T(), chartName, false, imageFlavor)
			})
		}
		s.Run(chartName, func() {
			testChartLint(s.T(), chartName, true, "")
		})
	}
}

func testChartLint(t *testing.T, chartName string, rhacs bool, imageFlavor string) {
	outputDir := t.TempDir()

	testIO, _, _, _ := io.TestIO()
	env := environment.NewTestCLIEnvironment(t, testIO, printer.DefaultColorPrinter())

	err := executeHelpOutputCommand(chartName, outputDir, true, imageFlavor, imageFlavor != "", rhacs, env)
	require.NoErrorf(t, err, "failed to output helm chart %s", chartName)

	for _, ns := range lintNamespaces {
		t.Run(fmt.Sprintf("namespace=%s", ns), func(t *testing.T) {
			testChartInNamespaceLint(t, outputDir, ns)
		})
	}
}

func executeHelpOutputCommand(chartName, outputDir string, removeOutputDir bool, imageFlavor string, flavorProvided, rhacs bool, env environment.Environment) error {
	cmd := helmOutputCommand{
		outputDir:       outputDir,
		removeOutputDir: removeOutputDir,
		rhacs:           rhacs,
		imageFlavor:     imageFlavor,
		env:             env,
	}

	c := &cobra.Command{Use: "test"}

	// Add and set `ImageDefaultsFlagName` flag if provided in the test scope
	if flavorProvided {
		c.Flags().Bool(flags.ImageDefaultsFlagName, flavorProvided, "")
		if err := c.Flags().Set(flags.ImageDefaultsFlagName, strconv.FormatBool(flavorProvided)); err != nil {
			return err
		}
	}

	// Execute command flow
	cmd.Construct(chartName, c)
	if err := cmd.Validate(); err != nil {
		return err
	}
	return cmd.outputHelmChart()
}

func testChartInNamespaceLint(t *testing.T, chartDir string, namespace string) {
	linter := lint.All(chartDir, nil, namespace, false)

	assert.LessOrEqualf(t, linter.HighestSeverity, maxTolerableSev, "linting chart produced warnings with severity %v", linter.HighestSeverity)
	for _, msg := range linter.Messages {
		fmt.Fprintln(os.Stderr, msg.Error())
		assert.LessOrEqual(t, msg.Severity, maxTolerableSev, msg.Error())
	}
}
