package charts

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
)

// TestRequiredMetaValuesArePresent validates that MetaValues attributes that are consumed and required by .htpl files
// are actually present.
func TestRequiredMetaValuesArePresent(t *testing.T) {
	testutils.SetExampleVersion(t)

	cases := []defaults.ImageFlavor{
		defaults.DevelopmentBuildImageFlavor(),
		defaults.StackRoxIOReleaseImageFlavor(),
		defaults.RHACSReleaseImageFlavor(),
		defaults.OpenSourceImageFlavor(),
	}
	for _, flavor := range cases {
		testName := fmt.Sprintf("Image Flavor %s", flavor.MainRegistry)
		t.Run(testName, func(t *testing.T) {
			metaVals := GetMetaValuesForFlavor(flavor)
			assert.NotEmpty(t, metaVals.MainRegistry)
			assert.NotEmpty(t, metaVals.ImageRemote)
			assert.NotEmpty(t, metaVals.CollectorRegistry)
			assert.NotEmpty(t, metaVals.CollectorFullImageRemote)
			assert.NotEmpty(t, metaVals.CollectorSlimImageRemote)
			assert.NotEmpty(t, metaVals.CollectorFullImageTag)
			assert.NotEmpty(t, metaVals.CollectorSlimImageTag)
			assert.NotEmpty(t, metaVals.ScannerImageRemote)
			assert.NotEmpty(t, metaVals.ScannerSlimImageRemote)
			assert.NotEmpty(t, metaVals.ScannerImageTag)
			assert.NotEmpty(t, metaVals.ScannerV4ImageRemote)
			assert.NotEmpty(t, metaVals.ScannerV4DBImageRemote)
			assert.NotEmpty(t, metaVals.ScannerV4ImageTag)
			assert.NotEmpty(t, metaVals.ChartRepo.URL)
			assert.NotNil(t, metaVals.ImagePullSecrets)

			assert.NotEmpty(t, metaVals.Versions.ChartVersion)
			assert.NotEmpty(t, metaVals.Versions.MainVersion)
			// TODO: replace this with the check of the scanner tag once we migrate to it instead of version.
			assert.NotEmpty(t, metaVals.Versions.ScannerVersion)
			assert.Len(t, metaVals.FeatureFlags, len(features.Flags))
		})
	}
}
