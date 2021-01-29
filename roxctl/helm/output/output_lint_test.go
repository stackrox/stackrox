package output

import (
	"fmt"
	"io/ioutil"
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
		t.Run(chartName, func(t *testing.T) {
			testChartLint(t, chartName)
		})
	}
}

func testChartLint(t *testing.T, chartName string) {
	outputDir, err := ioutil.TempDir("", "roxctl-helm-output-lint-")
	require.NoError(t, err)

	defer func() {
		_ = os.RemoveAll(outputDir)
	}()

	err = outputHelmChart(chartName, outputDir, true)
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
