package protocompat

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
)

func TestCompareTimestamps(t *testing.T) {
	protoTS1 := &types.Timestamp{
		Seconds: 2345678901,
		Nanos:   234567891,
	}

	protoTS2 := &types.Timestamp{
		Seconds: 3456789012,
		Nanos:   345678912,
	}

	assert.Zero(t, CompareTimestamps(protoTS1, protoTS1))
	assert.Negative(t, CompareTimestamps(protoTS1, protoTS2))
	assert.Positive(t, CompareTimestamps(protoTS2, protoTS1))
}

func TestConvertTimestampToCSVString(t *testing.T) {
	var nilTS *types.Timestamp
	nilStringRepresentation := ConvertTimestampToCSVString(nilTS)
	assert.Equal(t, "N/A", nilStringRepresentation)

	invalidTS := &types.Timestamp{
		Seconds: -62234567890,
	}
	stringFromInvalid := ConvertTimestampToCSVString(invalidTS)
	assert.Equal(t, "ERR", stringFromInvalid)

	ts := &types.Timestamp{
		Seconds: 2345678901,
		Nanos:   123456789,
	}
	timeString := ConvertTimestampToCSVString(ts)
	assert.Equal(t, "Sun, 01 May 2044 01:28:21 UTC", timeString)
}

func TestConvertTimestampToTimeOrError(t *testing.T) {
	seconds := int64(2345678901)
	nanos := int32(123456789)

	var protoTS1 *types.Timestamp
	protoTS2 := &types.Timestamp{
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
	assert.Equal(t, &types.Timestamp{Seconds: seconds1, Nanos: nanos1}, protoTS1)
}

func TestConvertTimestampToTimeOrNil(t *testing.T) {
	seconds := int64(2345678901)
	nanos := int32(123456789)

	var protoTS1 *types.Timestamp
	protoTS2 := &types.Timestamp{
		Seconds: seconds,
		Nanos:   nanos,
	}

	goTS1 := ConvertTimestampToTimeOrNil(protoTS1)
	assert.Nil(t, goTS1)

	expectedTimeTS2 := time.Unix(seconds, int64(nanos))

	goTS2 := ConvertTimestampToTimeOrNil(protoTS2)
	assert.Equal(t, expectedTimeTS2.Local(), goTS2.Local())

	protoTSInvalid := &types.Timestamp{
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
	assert.Equal(t, &types.Timestamp{Seconds: seconds1, Nanos: nanos1}, protoTS1)

	timeInvalid := time.Date(0, 12, 25, 23, 59, 59, 0, time.UTC)
	protoTSInvalid := ConvertTimeToTimestampOrNil(&timeInvalid)
	assert.Nil(t, protoTSInvalid)
}

func TestGetProtoTimestampFromRFC3339NanoString(t *testing.T) {
	timeString := "2017-11-16T19:35:32.012345678Z"

	ts1, err1 := GetProtoTimestampFromRFC3339NanoString(timeString)
	assert.NoError(t, err1)
	assert.Equal(t, int64(1510860932), ts1.Seconds)
	assert.Equal(t, int32(12345678), ts1.Nanos)

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
	var nilTS *types.Timestamp
	nowFromNil := NilOrNow(nilTS)
	assert.NotNil(t, nowFromNil)
	deltaFromNil := nowFromNil.Sub(now)
	// ensure the delta between "now" and the "now" from conversion is small enough
	assert.Equal(t, "0s", deltaFromNil.Truncate(time.Second).String())

	invalidTS := &types.Timestamp{
		Seconds: -62234567890,
	}
	nowFromInvalid := NilOrNow(invalidTS)
	assert.NotNil(t, nowFromInvalid)
	deltaFromInvalid := nowFromInvalid.Sub(now)
	// ensure the delta between "now" and the "now" from conversion is small enough
	assert.Equal(t, "0s", deltaFromInvalid.Truncate(time.Second).String())

	ts := &types.Timestamp{
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
	protoDuration := &types.Duration{
		Seconds: 1,
		Nanos:   5,
	}
	expectedDuration := 1000000005 * time.Nanosecond
	timeDuration, err := DurationFromProto(protoDuration)
	assert.NoError(t, err)
	assert.Equal(t, expectedDuration, timeDuration)
}

func TestDurationProto(t *testing.T) {
	timeDuration := 1000000005 * time.Nanosecond
	expectedProtoDuration := &types.Duration{
		Seconds: 1,
		Nanos:   5,
	}
	protoDuration := DurationProto(timeDuration)
	assert.Equal(t, expectedProtoDuration, protoDuration)
}
