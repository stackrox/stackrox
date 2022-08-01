package testbuildinfo

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/buildinfo/internal/timestamp"
	"github.com/stackrox/rox/pkg/testutils/utils"
)

// TestBuildTimestampRestorer restores previous build timestamp settings.
type TestBuildTimestampRestorer struct {
	prevTimestamp time.Time
	prevError     error
}

// Restore restores the previous build timestamp settings.
func (r *TestBuildTimestampRestorer) Restore() {
	if r == nil {
		return
	}
	timestamp.BuildTimestamp, timestamp.BuildTimestampParsingErr = r.prevTimestamp, r.prevError
}

// SetForTest sets the build timestamp to now if it is not currently set. This is exclusively intended to be used
// in test settings.
func SetForTest(t *testing.T) *TestBuildTimestampRestorer {
	utils.MustBeInTest(t)
	if timestamp.BuildTimestampParsingErr == nil {
		return nil // we have a valid build timestamp
	}
	restorer := TestBuildTimestampRestorer{
		prevTimestamp: timestamp.BuildTimestamp,
		prevError:     timestamp.BuildTimestampParsingErr,
	}
	timestamp.BuildTimestamp, timestamp.BuildTimestampParsingErr = time.Now(), nil
	return &restorer
}
