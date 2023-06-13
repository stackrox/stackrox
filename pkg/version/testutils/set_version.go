//go:build !release || test

package testutils

import (
	"runtime"
	"testing"

	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/version"
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

// SetExampleVersion sets the version to the example, only intended for usage in testing.
func SetExampleVersion(t *testing.T) {
	testutils.MustBeInTest(t)
	SetVersion(t, GetExampleVersion(t))
}

// SetVersion sets the version, only intended for usage in testing.
func SetVersion(t *testing.T, version version.Versions) {
	testutils.MustBeInTest(t)
	internal.MainVersion = version.MainVersion
	internal.ScannerVersion = version.ScannerVersion
	internal.CollectorVersion = version.CollectorVersion
	internal.GitShortSha = version.GitCommit
}

// GetExampleVersion returns an example version, only intended for usage in testing.
func GetExampleVersion(t *testing.T) version.Versions {
	testutils.MustBeInTest(t)
	return version.Versions{
		CollectorVersion: "99.9.9",
		GitCommit:        "45b4a8ac",
		GoVersion:        runtime.Version(),
		MainVersion:      "3.0.99.0",
		Platform:         runtime.GOOS + "/" + runtime.GOARCH,
		ScannerVersion:   "99.9.9",
		ChartVersion:     "3.99.0",
	}
}
