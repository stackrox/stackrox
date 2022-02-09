package testutils

import (
	"testing"

	"github.com/stackrox/rox/pkg/helm/charts"
	flavorUtils "github.com/stackrox/rox/pkg/images/defaults/testutils"
	"github.com/stackrox/rox/pkg/testutils"
)

// MakeMetaValuesForTest creates pre-populated charts.MetaValues for use in tests.
func MakeMetaValuesForTest(t *testing.T) *charts.MetaValues {
	testutils.MustBeInTest(t)
	return charts.GetMetaValuesForFlavor(flavorUtils.MakeImageFlavorForTest(t))
}
