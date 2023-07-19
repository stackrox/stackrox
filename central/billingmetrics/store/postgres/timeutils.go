package postgres

import (
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/pkg/protoconv"
)

func timeToTimestamp(t time.Time) *types.Timestamp {
	return protoconv.ConvertTimeToTimestamp(t.Truncate(time.Microsecond))
}

func timestampToStoreInUTC(ts *types.Timestamp) time.Time {
	return protoconv.ConvertTimestampToTimeOrNow(ts).Truncate(time.Microsecond).UTC()
}

func localTimestampFromStore(t *time.Time) *types.Timestamp {
	return protoconv.ConvertTimeToTimestamp(t.Local())
}
