package protocompat

import (
	"time"

	gogoTimestamp "github.com/gogo/protobuf/types"
)

// TimestampNow returns a protobuf timestamp set to the current time.
func TimestampNow() *gogoTimestamp.Timestamp {
	return gogoTimestamp.TimestampNow()
}

// ConvertTimestampToTimeOrError converts a proto timestamp to a golang Time, or returns an error if there is one.
func ConvertTimestampToTimeOrError(gogo *gogoTimestamp.Timestamp) (time.Time, error) {
	return gogoTimestamp.TimestampFromProto(gogo)
}

// ConvertTimeToTimestampOrError converts golang time to proto timestamp.
func ConvertTimeToTimestampOrError(goTime time.Time) (*gogoTimestamp.Timestamp, error) {
	return gogoTimestamp.TimestampProto(goTime)
}

// GetProtoTimestampFromSeconds instantiates a protobuf Timestamp structure initialized
// with the input to the seconds granularity
func GetProtoTimestampFromSeconds(seconds int64) *gogoTimestamp.Timestamp {
	return &gogoTimestamp.Timestamp{Seconds: seconds}
}

// CompareTimestamps compares two timestamps and returns zero if equal, a negative value if
// the first timestamp is before the second or a positive value if the first timestamp is
// after the second.
func CompareTimestamps(t1 *gogoTimestamp.Timestamp, t2 *gogoTimestamp.Timestamp) int {
	return t1.Compare(t2)
}
