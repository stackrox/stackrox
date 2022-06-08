package proto

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ValidateTSInWindow validates that the given proto timestamp lies in the window of the given
// Go timestamps.
func ValidateTSInWindow(ts *types.Timestamp, earliest, latest time.Time, t *testing.T) {
	asGoTime, err := types.TimestampFromProto(ts)
	require.NoError(t, err)

	ValidateTimeInWindow(asGoTime, earliest, latest, 0, t)
}

// ValidateTimeInWindow validates that the given time lies in the window of the given
// Go timestamps.
func ValidateTimeInWindow(time, earliest, latest time.Time, skewBuffer time.Duration, t *testing.T) {
	assert.True(t, time.After(earliest.Add(-skewBuffer)), "earliest: %s, got time: %s", earliest, time)
	assert.True(t, time.Before(latest.Add(skewBuffer)), "latest: %s, got time: %s", latest, time)
}
