package protocompat

import (
	"reflect"
	"time"

	"github.com/graph-gophers/graphql-go"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultTimeStringFormat = time.RFC3339Nano

var (
	// TimestampPtrType is a variable containing a nil pointer of Timestamp type
	TimestampPtrType = reflect.TypeOf((*timestamppb.Timestamp)(nil))

	// TimestampType is the type representing a proto timestamp.
	TimestampType = reflect.TypeOf(timestamppb.Timestamp{})
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
type Timestamp = timestamppb.Timestamp

// TimestampNow returns a protobuf timestamp set to the current time.
func TimestampNow() *timestamppb.Timestamp {
	return timestamppb.Now()
}

// ConvertTimestampToString converts a proto timestamp to a string.
func ConvertTimestampToString(timestamp *timestamppb.Timestamp, format string) string {
	if timestamp == nil {
		return "N/A"
	}
	if timestamp.CheckValid() != nil {
		return "ERR"
	}
	if format == "" {
		format = defaultTimeStringFormat
	}
	return timestamp.AsTime().Format(format)
}

// ConvertTimestampToTimeOrNil converts a proto timestamp to a golang Time, defaulting to nil in case of error.
func ConvertTimestampToTimeOrNil(pbTime *timestamppb.Timestamp) *time.Time {
	if pbTime == nil {
		return nil
	}
	goTime, err := ConvertTimestampToTimeOrError(pbTime)
	if err != nil {
		return nil
	}
	return &goTime
}

// ConvertTimeToTimestampOrNil converts a golang Time to a proto timestamp, defaulting to nil in case of error.
func ConvertTimeToTimestampOrNil(goTime *time.Time) *timestamppb.Timestamp {
	if goTime == nil {
		return nil
	}
	pbTime, err := ConvertTimeToTimestampOrError(*goTime)
	if err != nil {
		return nil
	}
	return pbTime
}

// ConvertTimestampToGraphqlTimeOrError converts a proto timestamp
// to a graphql Time, or returns an error if there is one.
func ConvertTimestampToGraphqlTimeOrError(pbTime *timestamppb.Timestamp) (*graphql.Time, error) {
	if pbTime == nil {
		return nil, nil
	}

	return &graphql.Time{Time: pbTime.AsTime()}, pbTime.CheckValid()
}

// ConvertTimestampToTimeOrError converts a proto timestamp
// to a golang Time, or returns an error if there is one.
func ConvertTimestampToTimeOrError(pbTime *timestamppb.Timestamp) (time.Time, error) {
	return pbTime.AsTime(), pbTime.CheckValid()
}

// ConvertTimeToTimestampOrError converts golang time to proto timestamp.
func ConvertTimeToTimestampOrError(goTime time.Time) (*timestamppb.Timestamp, error) {
	ts := timestamppb.New(goTime)
	if err := ts.CheckValid(); err != nil {
		return nil, err
	}

	return ts, nil
}

// GetProtoTimestampFromRFC3339NanoString generates a proto timestamp from a time string in RFC3339Nano format.
func GetProtoTimestampFromRFC3339NanoString(timeStr string) (*timestamppb.Timestamp, error) {
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
func GetProtoTimestampFromSeconds(seconds int64) *timestamppb.Timestamp {
	return &timestamppb.Timestamp{Seconds: seconds}
}

// GetProtoTimestampFromSecondsAndNanos instantiates a protobuf Timestamp structure initialized
// with the input to the seconds and nanoseconds granularity.
func GetProtoTimestampFromSecondsAndNanos(seconds int64, nanos int32) *timestamppb.Timestamp {
	return &timestamppb.Timestamp{Seconds: seconds, Nanos: nanos}
}

// GetProtoTimestampZero instantiates a protobuf Timestamp structure initialized
// with the zero values for all fields.
func GetProtoTimestampZero() *timestamppb.Timestamp {
	return &timestamppb.Timestamp{}
}

// NilOrNow allows for a proto timestamp to be stored a timestamp type in Postgres
func NilOrNow(t *timestamppb.Timestamp) *time.Time {
	now := time.Now()
	if t == nil {
		return &now
	}
	ts, err := ConvertTimestampToTimeOrError(t)
	if err != nil {
		return &now
	}
	ts = ts.Round(time.Microsecond)
	return &ts
}

// NilOrTime allows for a proto timestamp to be stored a timestamp type in Postgres
func NilOrTime(t *timestamppb.Timestamp) *time.Time {
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
func ParseRFC3339NanoTimestamp(timestamp string) (*timestamppb.Timestamp, error) {
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
func CompareTimestamps(t1 *timestamppb.Timestamp, t2 *timestamppb.Timestamp) int {
	return t1.AsTime().Compare(t2.AsTime())
}

// CompareTimestampToTime compares a proto timestamp to a time.
// The return value is:
// * -1 if the proto timestamp is before the time
// * 0 if both represent the same time
// * 1 if the proto timestamp is after the time
func CompareTimestampToTime(t1 *timestamppb.Timestamp, t2 *time.Time) int {
	ts2 := ConvertTimeToTimestampOrNil(t2)
	return CompareTimestamps(t1, ts2)
}

// DurationFromProto converts a proto Duration to a time.Duration.
//
// DurationFromProto returns an error if the Duration is invalid or is too large
// to be represented in a time.Duration.
func DurationFromProto(d *durationpb.Duration) (time.Duration, error) {
	// TODO: We didn't cover error case in unit tests!
	if err := d.CheckValid(); err != nil {
		return 0, err
	}

	return d.AsDuration(), nil
}

// DurationProto converts a time.Duration to a proto Duration.
func DurationProto(d time.Duration) *durationpb.Duration {
	return durationpb.New(d)
}

var (
	// zeroProtoTimestampFromTime represents the zero value of a proto
	// timestamp when initialized from the zero time.
	zeroProtoTimestampFromTime, _ = ConvertTimeToTimestampOrError(time.Time{})
)

// IsZeroTimestamp returns whether a Timestamp pointer is either nil, or pointing to the zero of the type.
func IsZeroTimestamp(ts *timestamppb.Timestamp) bool {
	return ts == nil ||
		CompareTimestamps(ts, GetProtoTimestampZero()) == 0 ||
		CompareTimestamps(ts, zeroProtoTimestampFromTime) == 0
}
