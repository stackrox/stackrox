package testutils

import (
	"testing"

	"github.com/stackrox/rox/pkg/helm/charts"
	flavorUtils "github.com/stackrox/rox/pkg/images/testutils"
	"github.com/stackrox/rox/pkg/testutils"
)

// DefaultTestMetaValues creates pre-populated charts.MetaValues for use in tests.
func DefaultTestMetaValues(t *testing.T) charts.MetaValues {
	testutils.MustBeInTest(t)
	return charts.GetMetaValuesForFlavor(flavorUtils.MakeImageFlavorForTest(t))
}
