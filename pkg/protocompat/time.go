package protocompat

import (
	"reflect"
	"time"

	gogoTimestamp "github.com/gogo/protobuf/types"
)

var (
	// TimestampPtrType is a variable containing a nil pointer of Timestamp type
	TimestampPtrType = reflect.TypeOf((*gogoTimestamp.Timestamp)(nil))
)

// TimestampNow returns a protobuf timestamp set to the current time.
func TimestampNow() *gogoTimestamp.Timestamp {
	return gogoTimestamp.TimestampNow()
}

// ConvertTimestampToTimeOrError converts a proto timestamp to a golang Time, or returns an error if there is one.
func ConvertTimestampToTimeOrError(gogo *gogoTimestamp.Timestamp) (time.Time, error) {
	return gogoTimestamp.TimestampFromProto(gogo)
}

// ConvertTimestampToTimeOrNil converts a proto timestamp to a golang Time, defaulting to nil in case of error.
func ConvertTimestampToTimeOrNil(gogo *gogoTimestamp.Timestamp) *time.Time {
	if gogo == nil {
		return nil
	}
	goTime, err := ConvertTimestampToTimeOrError(gogo)
	if err != nil {
		return nil
	}
	return &goTime
}

// ConvertTimeToTimestampOrNil converts a golang Time to a proto timestamp, defaulting to nil in case of error.
func ConvertTimeToTimestampOrNil(goTime *time.Time) *gogoTimestamp.Timestamp {
	if goTime == nil {
		return nil
	}
	gogo, err := ConvertTimeToTimestampOrError(*goTime)
	if err != nil {
		return nil
	}
	return gogo
}

// ConvertTimeToTimestampOrError converts golang time to proto timestamp.
func ConvertTimeToTimestampOrError(goTime time.Time) (*gogoTimestamp.Timestamp, error) {
	return gogoTimestamp.TimestampProto(goTime)
}

// GetProtoTimestampFromSeconds instantiates a protobuf Timestamp structure initialized
// with the input to the seconds granularity.
func GetProtoTimestampFromSeconds(seconds int64) *gogoTimestamp.Timestamp {
	return &gogoTimestamp.Timestamp{Seconds: seconds}
}

// GetProtoTimestampFromSecondsAndNanos instantiates a protobuf Timestamp structure initialized
// with the input to the seconds and nanoseconds granularity.
func GetProtoTimestampFromSecondsAndNanos(seconds int64, nanos int32) *gogoTimestamp.Timestamp {
	return &gogoTimestamp.Timestamp{Seconds: seconds, Nanos: nanos}
}

// GetProtoTimestampZero instantiates a protobuf Timestamp structure initialized
// with the zero values for all fields.
func GetProtoTimestampZero() *gogoTimestamp.Timestamp {
	return &gogoTimestamp.Timestamp{}
}

// NilOrTime allows for a proto timestamp to be stored a timestamp type in Postgres
func NilOrTime(t *gogoTimestamp.Timestamp) *time.Time {
	if t == nil {
		return nil
	}
	ts, err := ConvertTimestampToTimeOrError(t)
	if err != nil {
		return nil
	}
	ts = ts.Round(time.Microsecond)
	return &ts
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

var (
	// zeroProtoTimestampFromTime represents the zero value of a proto
	// timestamp when initialized from the zero time.
	zeroProtoTimestampFromTime, _ = ConvertTimeToTimestampOrError(time.Time{})
)

// IsZeroTimestamp returns whether a Timestamp pointer is either nil, or pointing to the zero of the type.
func IsZeroTimestamp(ts *gogoTimestamp.Timestamp) bool {
	return ts == nil || ts == &gogoTimestamp.Timestamp{} || ts == zeroProtoTimestampFromTime
}
