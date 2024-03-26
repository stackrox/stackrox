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
	TimestampType = reflect.TypeOf(gogoTimestamp.Timestamp{})
)

// Timestamp represents a point in time independent of any time zone or local calendar, encoded
// as a count of seconds and fractions of seconds at nanosecond resolution. The count is relative
// to an epoch at UTC midnight on January 1, 1970, in the proleptic Gregorian calendar which
// extends the Gregorian calendar backwards to year one.
//
// All minutes are 60 seconds long. Leap seconds are "smeared" so that no leap second table
// is needed for interpretation, using a
// [24-hour linear smear](https://developers.google.com/time/smear ).
//
// The range is from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59.999999999Z. By restricting to that
// range, we ensure that we can convert to and from
// [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt ) date strings.
type Timestamp = gogoTimestamp.Timestamp

// TimestampNow returns a protobuf timestamp set to the current time.
func TimestampNow() *gogoTimestamp.Timestamp {
	return gogoTimestamp.TimestampNow()
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

// GetProtoTimestampFromRFC3339NanoString generates a proto timestamp from a time string in RFC3339Nano format.
func GetProtoTimestampFromRFC3339NanoString(timeStr string) (*gogoTimestamp.Timestamp, error) {
	stringTime, err := time.Parse(time.RFC3339Nano, timeStr)
	if err != nil {
		return nil, err
	}
	timestamp, err := ConvertTimeToTimestampOrError(stringTime)
	if err != nil {
		return nil, err
	}
	return timestamp, nil
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

// ParseRFC3339NanoTimestamp converts a time string in RFC 3339 Nano format to a protobuf timestamp.
func ParseRFC3339NanoTimestamp(timestamp string) (*gogoTimestamp.Timestamp, error) {
	t, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return nil, err
	}
	protoTime, err := ConvertTimeToTimestampOrError(t)
	if err != nil {
		return nil, err
	}
	return protoTime, nil
}

// CompareTimestamps compares two timestamps and returns zero if equal, a negative value if
// the first timestamp is before the second or a positive value if the first timestamp is
// after the second.
func CompareTimestamps(t1 *gogoTimestamp.Timestamp, t2 *gogoTimestamp.Timestamp) int {
	return t1.Compare(t2)
}

// CompareTimestampToTime compares a proto timestamp to a time.
// The return value is:
// * -1 if the proto timestamp is before the time
// * 0 if both represent the same time
// * 1 if the proto timestamp is after the time
func CompareTimestampToTime(t1 *gogoTimestamp.Timestamp, t2 *time.Time) int {
	ts2 := ConvertTimeToTimestampOrNil(t2)
	return CompareTimestamps(t1, ts2)
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
	return ts == nil || Equal(ts, GetProtoTimestampZero()) || Equal(ts, zeroProtoTimestampFromTime)
}
