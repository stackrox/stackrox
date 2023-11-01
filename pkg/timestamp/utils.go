package timestamp

import (
	"time"

	"github.com/gogo/protobuf/types"
	protoTypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/pkg/protoconv"
)

// RoundTimestamp rounds up ts to the nearest multiple of d. In case of error, the function returns without rounding up.
func RoundTimestamp(ts *types.Timestamp, d time.Duration) {
	t, err := types.TimestampFromProto(ts)
	if err != nil {
		return
	}
	*ts = *protoconv.ConvertTimeToTimestamp(t.Round(d))
}

// NowMinus substracts a specified amount of time from the current timestamp
func NowMinus(t time.Duration) *types.Timestamp {
	return protoconv.ConvertTimeToTimestamp(time.Now().Add(-t))
}

// TimeBeforeDays subtracts a specified number of days from the current timestamp
func TimeBeforeDays(days int) *protoTypes.Timestamp {
	return NowMinus(24 * time.Duration(days) * time.Hour)
}
