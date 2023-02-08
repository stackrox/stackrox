package buildinfo

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/buildinfo/internal/timestamp"
)

const (
	// ReleaseBuild indicates whether this build is a release build.
	ReleaseBuild bool = releaseBuild

	// BuildFlavor indicates the build flavor ("release" for release builds, "development" for development builds).
	BuildFlavor string = buildFlavor

	// RaceEnabled indicates whether the build was created with the race detector enabled. This usually only applies to
	// tests, and will be false for actual binary builds.
	RaceEnabled = raceEnabled
)

// BuildTimestamp returns the time when this build was created.
// CAVEAT: This function panics if no build timestamp information is available.
//
// Deprecated: It will be removed in 4.0. Please do not use it.
// TODO(ROX-14336): delete it
func BuildTimestamp() time.Time {
	if timestamp.BuildTimestampParsingErr != nil {
		panic(errors.Wrap(timestamp.BuildTimestampParsingErr, "failed to parse build timestamp"))
	}
	return timestamp.BuildTimestamp
}
