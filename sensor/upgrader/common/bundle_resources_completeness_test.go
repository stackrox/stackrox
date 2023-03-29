package common

import (
	"testing"

	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/helm/util"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// gvksExcludedFromCompletenessTest are the GVKs that may be present in the bundle without being in the
	// OrderedBundleResourceTypes list. We allow these because they are guarded by configuration options which are
	// unavailable for bundle-based deployment (upgrader).
	gvksExcludedFromCompletenessTest = []schema.GroupVersionKind{}
)

// TestBundleResourcesComplete tests that every resource that *could* appear in the Sensor bundle is accounted for
// in the OrderedBundleResourceTypes list.
// The set of resources that could appear is extracted from the Helm chart in a heuristic manner. Sometimes, the
// set of potential resources contains false positives.
// If you see this test failing, the right course of action is most likely to extend the OrderedBundleResourceTypes
// list. However, if you are convinced that one of the reported GVKs is a false positive, you have the following
// options:
//   - If the GVK in question can never appear in YAML bundle mode, use meta-templating ([< if not .KubectlOutput >])
//     blocks to exclude the resources from the chart.
//   - If the GVK is reported due to the approximative nature of the GVK from chart extractor, often small reformatting
//     of the chart can be helpful. It's often sufficient to include an extra "---" document separator before the
//     start of the object definition.
//   - If the false positive cannot be suppressed using the above methods, you can add it to the above
//     ExcludedFromCompletenessTest list to explicitly suppress it in this test.
func TestBundleResourcesComplete(t *testing.T) {
	featureFlags := make(map[string]interface{})
	for _, ff := range features.Flags {
		featureFlags[ff.EnvVar()] = ff.Enabled()
	}
	metaValues := &charts.MetaValues{
		Versions: version.Versions{
			ChartVersion:     "1.0.0",
			MainVersion:      "3.0.49.0",
			CollectorVersion: "1.2.3",
		},
		MainRegistry:             "stackrox.io", // TODO: custom?
		ImageRemote:              "main",
		ImageTag:                 "3.0.49.0",
		CollectorRegistry:        "collector.stackrox.io",
		CollectorFullImageRemote: "collector",
		CollectorSlimImageRemote: "collector",
		CollectorSlimImageTag:    "1.2.3-slim",
		CollectorFullImageTag:    "1.2.3",
		ScannerImageRemote:       "scanner",
		ScannerSlimImageRemote:   "scanner",
		ScannerImageTag:          "1.2.3",
		ChartRepo: defaults.ChartRepo{
			URL:     "http://mirror.openshift.com/pub/rhacs/charts",
			IconURL: "https://raw.githubusercontent.com/stackrox/stackrox/master/image/templates/helm/shared/assets/Red_Hat-Hat_icon.png",
		},
		KubectlOutput: true,
		FeatureFlags:  featureFlags,
	}

	helmImage := image.GetDefaultImage()
	tpl, err := helmImage.GetSecuredClusterServicesChartTemplate()
	require.NoError(t, err, "error retrieving chart template")
	ch, err := tpl.InstantiateAndLoad(metaValues)
	require.NoError(t, err, "error instantiating chart")

	gvksInChart, err := util.ExtractApproximateGVKsFromChart(ch)
	require.NoError(t, err)

	chartGVKsNotInBundleResource := make(map[schema.GroupVersionKind]struct{})
	for _, gvk := range gvksInChart {
		chartGVKsNotInBundleResource[gvk] = struct{}{}
	}

	for _, gvk := range OrderedBundleResourceTypes {
		delete(chartGVKsNotInBundleResource, gvk)
	}

	for _, gvk := range gvksExcludedFromCompletenessTest {
		delete(chartGVKsNotInBundleResource, gvk)
	}

	assert.Empty(t, chartGVKsNotInBundleResource, "some GVKs that might occur as part of the sensor bundle are not accounted for in the upgrader code")
}
