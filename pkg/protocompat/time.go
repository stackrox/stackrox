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

// DurationFromProto converts a proto Duration to a time.Duration.
//
// DurationFromProto returns an error if the Duration is invalid or is too large
// to be represented in a time.Duration.
func DurationFromProto(d *gogoTimestamp.Duration) (time.Duration, error) {
	return gogoTimestamp.DurationFromProto(d)
}

// DurationProto converts a time.Duration to a proto Duration.
func DurationProto(d time.Duration) *gogoTimestamp.Duration {
	return gogoTimestamp.DurationProto(d)
}
