// +build !release

package testutils

import (
	"testing"

	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/version/internal"
)

// SetMainVersion sets the main version to the given string.
// IT IS ONLY INTENDED FOR USE IN TESTING.
// It will NOT work in release builds anyway because this code is excluded
// by build constraints.
// To make this more explicit, we require passing a testing.T to this version.
func SetMainVersion(t *testing.T, version string) {
	testutils.MustBeInTest(t)
	internal.MainVersion = version
}
