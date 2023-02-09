package testutils

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/buildinfo/internal/timestamp"
)

// SetBuildTimestamp sets the build timestamp in UNIX secs. This function is only intended for testing.
func SetBuildTimestamp(t *testing.T, buildTimestamp time.Time) {
	utils.MustBeInTest(t)
	timestamp.BuildTimestamp = buildTimestamp
	timestamp.BuildTimestampParsingErr = nil
}
