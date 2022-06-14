package testutils

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/buildinfo/internal/timestamp"
	"github.com/stackrox/rox/pkg/testutils"
)

// SetBuildTimestamp sets the build timestamp in UNIX secs. This function is only intended for testing.
func SetBuildTimestamp(t *testing.T, buildTimestamp time.Time) {
	testutils.MustBeInTest(t)
	timestamp.BuildTimestamp = buildTimestamp
	timestamp.BuildTimestampParsingErr = nil
}
