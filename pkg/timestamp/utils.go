package timestamp

import (
	"time"

	"github.com/gogo/protobuf/types"
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
