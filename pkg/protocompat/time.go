package protocompat

import (
	"reflect"
	"time"

	gogoTimestamp "github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
)

var (
	// TimestampPtrType is a variable containing a nil pointer of Timestamp type
	TimestampPtrType = reflect.TypeOf((*gogoTimestamp.Timestamp)(nil))

	// TimestampType is the type representing a proto timestamp.
	var TimestampType = reflect.TypeOf(gogoTimestamp.Timestamp{})

	// TimestampPointerType is the type representing a proto timestamp.
	var TimestampPointerType = reflect.TypeOf((*gogoTimestamp.Timestamp)(nil))
)

// TimestampNow returns a protobuf timestamp set to the current time.
func TimestampNow() *gogoTimestamp.Timestamp {
	return gogoTimestamp.TimestampNow()
}

// ConvertTimestampToGraphqlTimeOrError converts a proto timestamp
// to a graphql Time, or returns an error if there is one.
func ConvertTimestampToGraphqlTimeOrError(gogo *gogoTimestamp.Timestamp) (*graphql.Time, error) {
	if gogo == nil {
		return nil, nil
	}
	t, err := gogoTimestamp.TimestampFromProto(gogo)
	return &graphql.Time{Time: t}, err
}

// ConvertTimestampToTimeOrError converts a proto timestamp
// to a golang Time, or returns an error if there is one.
func ConvertTimestampToTimeOrError(gogo *gogoTimestamp.Timestamp) (time.Time, error) {
	return gogoTimestamp.TimestampFromProto(gogo)
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
