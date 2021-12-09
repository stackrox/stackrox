package tests

import (
	"testing"

	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/version"
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
		"Versions": version.Versions{
			ChartVersion:     "1.0.0",
			MainVersion:      "3.0.49.0",
			ScannerVersion:   "1.2.3",
			CollectorVersion: "3.4.0",
		},
		"FeatureFlags": map[string]interface{}{},
		"RenderMode":   "",
		"Operator":     false,
	}
}
