package output

import (
	"fmt"
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stackrox/rox/roxctl/helm/internal/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/lint"
	"helm.sh/helm/v3/pkg/lint/support"
)

const (
	maxTolerableSev = support.WarningSev
)

var (
	lintNamespaces = []string{"default", "stackrox"}
)

func init() {
	testutils.SetMainVersion(&testing.T{}, "3.0.55.0")
	testbuildinfo.SetForTest(&testing.T{})
}

func TestHelmLint(t *testing.T) {
	for chartName := range common.ChartTemplates {
		for _, rhacs := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s-rhacs-%v", chartName, rhacs), func(t *testing.T) {
				testChartLint(t, chartName, rhacs)
			})
		}
	}
}

func testChartLint(t *testing.T, chartName string, rhacs bool) {
	const noDebug = false
	const noDebugChartPath = ""
	outputDir, err := os.MkdirTemp("", "roxctl-helm-output-lint-")
	require.NoError(t, err)

	defer func() {
		_ = os.RemoveAll(outputDir)
	}()

	err = outputHelmChart(chartName, outputDir, true, rhacs, noDebug, noDebugChartPath)
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
