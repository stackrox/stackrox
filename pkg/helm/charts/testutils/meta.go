package testutils

import (
	"testing"

	"github.com/stackrox/stackrox/pkg/helm/charts"
	flavorUtils "github.com/stackrox/stackrox/pkg/images/defaults/testutils"
	"github.com/stackrox/stackrox/pkg/testutils"
)

// MakeMetaValuesForTest creates pre-populated charts.MetaValues for use in tests.
func MakeMetaValuesForTest(t *testing.T) *charts.MetaValues {
	testutils.MustBeInTest(t)
	return charts.GetMetaValuesForFlavor(flavorUtils.MakeImageFlavorForTest(t))
}
