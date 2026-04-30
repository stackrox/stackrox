package protocompat

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCompareTimestamps(t *testing.T) {
	protoTS1 := &timestamppb.Timestamp{
		Seconds: 2345678901,
		Nanos:   234567891,
	}

	protoTS2 := &timestamppb.Timestamp{
		Seconds: 3456789012,
		Nanos:   345678912,
	}

	assert.Zero(t, CompareTimestamps(protoTS1, protoTS1))
	assert.Negative(t, CompareTimestamps(protoTS1, protoTS2))
	assert.Positive(t, CompareTimestamps(protoTS2, protoTS1))
}

func TestConvertTimestampToString(t *testing.T) {
	var nilTS *timestamppb.Timestamp
	nilStringRepresentation := ConvertTimestampToString(nilTS, time.RFC3339Nano)
	assert.Equal(t, "N/A", nilStringRepresentation)

	invalidTS := &timestamppb.Timestamp{
		Seconds: -62234567890,
	}
	stringFromInvalid := ConvertTimestampToString(invalidTS, time.RFC3339Nano)
	assert.Equal(t, "ERR", stringFromInvalid)

	ts := &timestamppb.Timestamp{
		Seconds: 2345678901,
		Nanos:   123456789,
	}
	timeString := ConvertTimestampToString(ts, time.RFC3339)
	assert.Equal(t, "2044-05-01T01:28:21Z", timeString)

	timeStringNoFormatProvided := ConvertTimestampToString(ts, "")
	timeStringDefaultFormat := ConvertTimestampToString(ts, defaultTimeStringFormat)
	assert.Equal(t, timeStringDefaultFormat, timeStringNoFormatProvided)

	timeStringRFC1123 := ConvertTimestampToString(ts, time.RFC1123)
	assert.Equal(t, "Sun, 01 May 2044 01:28:21 UTC", timeStringRFC1123)
}

func TestConvertTimestampToTimeOrError(t *testing.T) {
	seconds := int64(2345678901)
	nanos := int32(123456789)

	var protoTS1 *timestamppb.Timestamp
	protoTS2 := &timestamppb.Timestamp{
		Seconds: seconds,
		Nanos:   nanos,
	}

	_, errTS1 := ConvertTimestampToTimeOrError(protoTS1)
	assert.Error(t, errTS1)

	expectedTimeTS2 := time.Unix(seconds, int64(nanos))

	timeTS2, errTS2 := ConvertTimestampToTimeOrError(protoTS2)
	assert.NoError(t, errTS2)
	assert.Equal(t, expectedTimeTS2.Local(), timeTS2.Local())
}

func TestConvertTimeToTimestampOrError(t *testing.T) {
	seconds1 := int64(2345678901)
	nanos1 := int32(123456789)
	time1 := time.Unix(seconds1, int64(nanos1))

	protoTS1, errTS1 := ConvertTimeToTimestampOrError(time1)
	assert.NoError(t, errTS1)
	assert.Equal(t, (&timestamppb.Timestamp{Seconds: seconds1, Nanos: nanos1}).AsTime(), protoTS1.AsTime())
}

func TestConvertTimestampToTimeOrNil(t *testing.T) {
	seconds := int64(2345678901)
	nanos := int32(123456789)

	var protoTS1 *timestamppb.Timestamp
	protoTS2 := &timestamppb.Timestamp{
		Seconds: seconds,
		Nanos:   nanos,
	}

	goTS1 := ConvertTimestampToTimeOrNil(protoTS1)
	assert.Nil(t, goTS1)

	expectedTimeTS2 := time.Unix(seconds, int64(nanos))

	goTS2 := ConvertTimestampToTimeOrNil(protoTS2)
	assert.Equal(t, expectedTimeTS2.Local(), goTS2.Local())

	protoTSInvalid := &timestamppb.Timestamp{
		Seconds: -62135683200,
	}
	goTSInvalid := ConvertTimestampToTimeOrNil(protoTSInvalid)
	assert.Nil(t, goTSInvalid)
}

func TestConvertTimeToTimestampOrNil(t *testing.T) {
	var timeNil *time.Time
	protoTSNil := ConvertTimeToTimestampOrNil(timeNil)
	assert.Nil(t, protoTSNil)

	seconds1 := int64(2345678901)
	nanos1 := int32(123456789)
	time1 := time.Unix(seconds1, int64(nanos1))

	protoTS1 := ConvertTimeToTimestampOrNil(&time1)
	assert.Equal(t, (&timestamppb.Timestamp{Seconds: seconds1, Nanos: nanos1}).AsTime(), protoTS1.AsTime())

	timeInvalid := time.Date(0, 12, 25, 23, 59, 59, 0, time.UTC)
	protoTSInvalid := ConvertTimeToTimestampOrNil(&timeInvalid)
	assert.Nil(t, protoTSInvalid)
}

func TestConvertTimestampToGraphqlTimeOrError(t *testing.T) {
	gqlTime, err := ConvertTimestampToGraphqlTimeOrError(nil)
	assert.NoError(t, err)
	assert.Nil(t, gqlTime)

	protoTime := &timestamppb.Timestamp{
		Seconds: 2345678901,
		Nanos:   234567891,
	}
	gqlTime, err = ConvertTimestampToGraphqlTimeOrError(protoTime)
	assert.NoError(t, err)
	assert.Equal(t, int64(2345678901234567891), gqlTime.UnixNano())

	// Negative Nanos is not valid for Timestamp.
	invalidProtoTime := &timestamppb.Timestamp{
		Nanos: -1,
	}
	gqlTime, err = ConvertTimestampToGraphqlTimeOrError(invalidProtoTime)
	assert.Error(t, err)
	assert.NotNil(t, gqlTime)
}

func TestGetProtoTimestampFromRFC3339NanoString(t *testing.T) {
	timeString := "2017-11-16T19:35:32.012345678Z"

	ts1, err1 := GetProtoTimestampFromRFC3339NanoString(timeString)
	assert.NoError(t, err1)
	assert.Equal(t, int64(1510860932), ts1.GetSeconds())
	assert.Equal(t, int32(12345678), ts1.GetNanos())

	invalidTimeString1 := "0000-12-24T23:59:59.999999999Z"
	_, err2 := GetProtoTimestampFromRFC3339NanoString(invalidTimeString1)
	assert.Error(t, err2)

	invalidTimeString2 := "0000-12-2AT23:59:59.999999999Z"
	_, err3 := GetProtoTimestampFromRFC3339NanoString(invalidTimeString2)
	assert.Error(t, err3)
}

func TestGetProtoTimestampFromSeconds(t *testing.T) {
	seconds1 := int64(1234567890)
	seconds2 := int64(23456789012)
	ts1 := GetProtoTimestampFromSeconds(seconds1)
	assert.Equal(t, seconds1, ts1.GetSeconds())
	assert.Equal(t, int32(0), ts1.GetNanos())
	ts2 := GetProtoTimestampFromSeconds(seconds2)
	assert.Equal(t, seconds2, ts2.GetSeconds())
	assert.Equal(t, int32(0), ts2.GetNanos())
}

func TestGetProtoTimestampFromSecondsAndNanos(t *testing.T) {
	seconds1 := int64(1234567890)
	nanos1 := int32(123456789)
	seconds2 := int64(23456789012)
	nanos2 := int32(234567890)
	ts1 := GetProtoTimestampFromSecondsAndNanos(seconds1, nanos1)
	assert.Equal(t, seconds1, ts1.GetSeconds())
	assert.Equal(t, nanos1, ts1.GetNanos())
	ts2 := GetProtoTimestampFromSecondsAndNanos(seconds2, nanos2)
	assert.Equal(t, seconds2, ts2.GetSeconds())
	assert.Equal(t, nanos2, ts2.GetNanos())
}

func TestGetProtoTimestampZero(t *testing.T) {
	ts1 := GetProtoTimestampZero()
	assert.Equal(t, int64(0), ts1.GetSeconds())
	assert.Equal(t, int32(0), ts1.GetNanos())
}

func TestNilOrNow(t *testing.T) {
	now := time.Now()
	var nilTS *timestamppb.Timestamp
	nowFromNil := NilOrNow(nilTS)
	assert.NotNil(t, nowFromNil)
	deltaFromNil := nowFromNil.Sub(now)
	// ensure the delta between "now" and the "now" from conversion is small enough
	assert.Equal(t, "0s", deltaFromNil.Truncate(time.Second).String())

	invalidTS := &timestamppb.Timestamp{
		Seconds: -62234567890,
	}
	nowFromInvalid := NilOrNow(invalidTS)
	assert.NotNil(t, nowFromInvalid)
	deltaFromInvalid := nowFromInvalid.Sub(now)
	// ensure the delta between "now" and the "now" from conversion is small enough
	assert.Equal(t, "0s", deltaFromInvalid.Truncate(time.Second).String())

	ts := &timestamppb.Timestamp{
		Seconds: int64(2345678901),
		Nanos:   int32(123456789),
	}
	timeFromTS := NilOrNow(ts)
	assert.NotNil(t, timeFromTS)
	assert.Equal(t, time.Date(2044, 5, 1, 1, 28, 21, 123457000, time.UTC), *timeFromTS)
}

func TestTimestampNow(t *testing.T) {
	nowTime := time.Now()
	nowTimestamp := TimestampNow()

	timeFromTimestamp, convertErr := ConvertTimestampToTimeOrError(nowTimestamp)
	assert.NoError(t, convertErr)
	timeDelta := timeFromTimestamp.Sub(nowTime)
	assert.Less(t, timeDelta, 500*time.Millisecond)
}

func TestDurationFromProto(t *testing.T) {
	protoDuration := &durationpb.Duration{
		Seconds: 1,
		Nanos:   5,
	}
	expectedDuration := 1000000005 * time.Nanosecond
	timeDuration, err := DurationFromProto(protoDuration)
	assert.NoError(t, err)
	assert.Equal(t, expectedDuration, timeDuration)

	// Positive duration with negative Nanos is not valid.
	protoInvalidDuration := &durationpb.Duration{
		Seconds: 1,
		Nanos:   -1,
	}
	timeDuration, err = DurationFromProto(protoInvalidDuration)
	assert.Error(t, err)
	assert.Equal(t, time.Duration(0), timeDuration)
}

func TestDurationProto(t *testing.T) {
	timeDuration := 1000000005 * time.Nanosecond
	expectedProtoDuration := &durationpb.Duration{
		Seconds: 1,
		Nanos:   5,
	}
	protoDuration := DurationProto(timeDuration)
	assert.Equal(t, expectedProtoDuration.AsDuration(), protoDuration.AsDuration())
}

// FuzzParseRFC3339NanoTimestamp tests that ParseRFC3339NanoTimestamp never panics
// on arbitrary string inputs and that valid RFC3339Nano timestamps round-trip correctly.
func FuzzParseRFC3339NanoTimestamp(f *testing.F) {
	// Seed corpus with valid RFC3339Nano timestamps
	seeds := []string{
		// Standard formats
		"2006-01-02T15:04:05.999999999Z",
		"2023-01-01T00:00:00Z",
		"2044-05-01T01:28:21.123456789Z",
		"2017-11-16T19:35:32.012345678Z",

		// Boundary cases
		"0001-01-01T00:00:00Z",           // Minimum valid time
		"9999-12-31T23:59:59.999999999Z", // Maximum valid time
		"1970-01-01T00:00:00Z",           // Unix epoch

		// Different nanosecond precisions
		"2023-06-15T12:30:45Z",           // No fractional seconds
		"2023-06-15T12:30:45.1Z",         // 1 digit
		"2023-06-15T12:30:45.12Z",        // 2 digits
		"2023-06-15T12:30:45.123Z",       // 3 digits (milliseconds)
		"2023-06-15T12:30:45.123456Z",    // 6 digits (microseconds)
		"2023-06-15T12:30:45.123456789Z", // 9 digits (nanoseconds)

		// Timezone offsets (RFC3339 allows these)
		"2023-06-15T12:30:45+00:00",
		"2023-06-15T12:30:45-07:00",
		"2023-06-15T12:30:45.123456789+05:30",

		// Edge cases from existing tests
		"0000-12-24T23:59:59.999999999Z", // Year 0 (invalid)
		"0000-12-2AT23:59:59.999999999Z", // Invalid day

		// Common invalid formats to test error handling
		"",
		"not a timestamp",
		"2023-13-01T00:00:00Z", // Invalid month
		"2023-01-32T00:00:00Z", // Invalid day
		"2023-01-01T25:00:00Z", // Invalid hour
		"2023-01-01",           // Missing time
		"12:30:45",             // Missing date
		"2023/01/01T12:30:45Z", // Wrong separator
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// Primary goal: ensure no panics
		ts, err := ParseRFC3339NanoTimestamp(input)

		if err != nil {
			// Error is acceptable, just ensure timestamp is nil
			assert.Nil(t, ts, "timestamp should be nil when error is returned")
			return
		}

		// If parsing succeeded, validate the result
		assert.NotNil(t, ts, "timestamp should not be nil when no error")

		// Ensure the timestamp is valid
		assert.NoError(t, ts.CheckValid(), "parsed timestamp should be valid")

		// Round-trip check: convert back to string and parse again
		goTime := ts.AsTime()
		roundTrip := goTime.Format(time.RFC3339Nano)
		ts2, err2 := ParseRFC3339NanoTimestamp(roundTrip)
		assert.NoError(t, err2, "round-trip parsing should succeed")
		assert.NotNil(t, ts2, "round-trip timestamp should not be nil")

		// Times should be equal (comparing as time.Time for precision handling)
		assert.Equal(t, goTime, ts2.AsTime(), "round-trip should preserve time")
	})
}
