package protoutils

import (
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/pkg/protocompat"
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

// RoundTimestamp rounds up ts to the nearest multiple of d. In case of error, the function returns without rounding up.
func RoundTimestamp(ts *types.Timestamp, d time.Duration) *types.Timestamp {
	t, err := protocompat.ConvertTimestampToTimeOrError(ts)
	if err != nil {
		return ts
	}
	return protoconv.ConvertTimeToTimestamp(t.Round(d))
}
