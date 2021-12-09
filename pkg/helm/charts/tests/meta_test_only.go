package tests

import (
	"testing"

	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/testutils"
	versionTestutils "github.com/stackrox/rox/pkg/version/testutils"
)

// DefaultTestMetaValues creates pre-populated charts.MetaValues for use in tests.
func DefaultTestMetaValues(t *testing.T) charts.MetaValues {
	testutils.MustBeInTest(t)
	return charts.MetaValues{
		"MainRegistry":          "stackrox.io",
		"CollectorRegistry":     "collector.stackrox.io",
		"CollectorFullImageTag": "3.4.0-latest",
		"CollectorSlimImageTag": "3.4.0-slim",
		"ChartRepo": charts.ChartRepo{
			URL: "https://charts.stackrox.io",
		},
		"ImagePullSecrets": charts.ImagePullSecrets{
			AllowNone: true,
		},
		"Versions":     versionTestutils.GetExampleVersion(t),
		"FeatureFlags": map[string]interface{}{},
		"RenderMode":   "",
		"Operator":     false,
	}
}
