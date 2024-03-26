package protoutils

import (
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/protoconv"
)

const (
	secondInt64 = int64(time.Second)
)

// Sub returns the difference between two timestamps
func Sub(ts1, ts2 *types.Timestamp) time.Duration {
	if ts1 == nil || ts2 == nil {
		return 0
	}
	seconds := ts1.GetSeconds() - ts2.GetSeconds()
	nanos := int64(ts1.GetNanos() - ts2.GetNanos())

	return time.Duration(seconds*secondInt64 + nanos)
}

// After returns whether the ts1 is after ts2.
func After(ts1, ts2 *types.Timestamp) bool {
	diff := Sub(ts1, ts2)
	return diff > 0
}

// MustGetProtoTimestampFromRFC3339NanoString generates a proto timestamp from a time string in RFC3339Nano format.
// The function panics if an error is raised in the conversion process.
func MustGetProtoTimestampFromRFC3339NanoString(timeStr string) *types.Timestamp {
	timestamp, err := protocompat.GetProtoTimestampFromRFC3339NanoString(timeStr)
	utils.CrashOnError(err)
	return timestamp
}

// NowMinus substracts a specified amount of time from the current timestamp
func NowMinus(t time.Duration) *types.Timestamp {
	return protoconv.ConvertTimeToTimestamp(time.Now().Add(-t))
}

// TimeBeforeDays subtracts a specified number of days from the current timestamp
func TimeBeforeDays(days int) *types.Timestamp {
	return NowMinus(24 * time.Duration(days) * time.Hour)
}

// RoundTimestamp rounds up ts to the nearest multiple of d. In case of error, the function returns without rounding up.
func RoundTimestamp(ts *types.Timestamp, d time.Duration) {
	t, err := protocompat.ConvertTimestampToTimeOrError(ts)
	if err != nil {
		return
	}
	*ts = *protoconv.ConvertTimeToTimestamp(t.Round(d))
}
