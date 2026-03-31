//go:build operator_helmtest

package chart

import (
	"testing"

	helmTest "github.com/stackrox/helmtest/pkg/framework"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
)

func TestWithHelmtest(t *testing.T) {
	suite, err := helmTest.NewLoader("testdata/helmtest").LoadSuite()
	require.NoError(t, err, "failed to load helmtest suite")
	ch, err := loader.Load("../../dist/chart")
	require.NoError(t, err, "failed to load chart")
	target := &helmTest.Target{
		Chart: ch,
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      "rhacs-operator",
			Namespace: "rhacs-operator-system",
			IsInstall: true,
		},
	}

	suite.Run(t, target)
}
