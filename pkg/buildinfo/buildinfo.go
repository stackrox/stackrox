package buildinfo

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/buildinfo/internal/timestamp"
)

const (
	// ReleaseBuild indicates whether this build is a release build.
	ReleaseBuild bool = releaseBuild

	// BuildFlavor indicates the build flavor ("release" for release builds, "development" for development builds).
	BuildFlavor string = buildFlavor
)

// BuildTimestamp returns the time when this build was created.
// CAVEAT: This function panics if no build timestamp information is available.
func BuildTimestamp() time.Time {
	if timestamp.BuildTimestampParsingErr != nil {
		panic(fmt.Errorf("failed to parse build timestamp: %v", timestamp.BuildTimestampParsingErr))
	}
	return timestamp.BuildTimestamp
}
