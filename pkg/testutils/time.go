package testutils

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
	assert.True(t, asGoTime.After(earliest), "earliest: %s, got time: %s", earliest, asGoTime)
	assert.True(t, asGoTime.Before(latest), "latest: %s, got time: %s", latest, asGoTime)

}
