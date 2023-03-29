package testutils

import (
	"encoding/json"
	"testing"

	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlavorsDontHaveEmptyFields(t *testing.T) {
	testutils.SetExampleVersion(t)

	flavors := []defaults.ImageFlavor{
		defaults.DevelopmentBuildImageFlavor(),
		defaults.StackRoxIOReleaseImageFlavor(),
		defaults.RHACSReleaseImageFlavor(),
		defaults.OpenSourceImageFlavor(),
		MakeImageFlavorForTest(t),
	}

	for _, f := range flavors {
		data, err := json.Marshal(f)
		require.NoError(t, err)

		mapData := make(map[string]interface{})
		err = json.Unmarshal(data, &mapData)
		require.NoError(t, err)

		for k, v := range mapData {
			assert.NotEmpty(t, v, "Field %q was empty but should have a value.", k)
		}
	}
}
