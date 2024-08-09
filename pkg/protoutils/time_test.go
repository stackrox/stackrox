package protoutils

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSub(t *testing.T) {
	now := time.Now()
	newTime := now.Add(time.Minute)

	nowTS := protoconv.ConvertTimeToTimestamp(now)
	newTS := protoconv.ConvertTimeToTimestamp(newTime)

	assert.Equal(t, 1*time.Minute, Sub(newTS, nowTS))
}

func TestAfter(t *testing.T) {
	now := time.Now()
	before := now.Add(-1 * time.Second)
	rightBefore := now.Add(-1 * time.Nanosecond)
	after := now.Add(1 * time.Second)
	rightAfter := now.Add(1 * time.Nanosecond)

	nowTs := protoconv.ConvertTimeToTimestamp(now)
	beforeTs := protoconv.ConvertTimeToTimestamp(before)
	rightBeforeTs := protoconv.ConvertTimeToTimestamp(rightBefore)
	afterTs := protoconv.ConvertTimeToTimestamp(after)
	rightAfterTs := protoconv.ConvertTimeToTimestamp(rightAfter)

	assert.False(t, After(beforeTs, nowTs), "After() should return false for a time that is 1s before now")
	assert.False(t, After(rightBeforeTs, nowTs), "After() should return false for a time that is 1ns before now")
	assert.False(t, After(nowTs, nowTs), "After() should return false for a time that is same as now")

	assert.True(t, After(afterTs, nowTs), "After() should return true for a time that is 1s after now")
	assert.True(t, After(rightAfterTs, nowTs), "After() should return true for a time that is 1ns after now")
}

func TestMustGetProtoTimestampFromRFC3339NanoString(t *testing.T) {
	timeString := "2017-11-16T19:35:32.012345678Z"

	ts1 := MustGetProtoTimestampFromRFC3339NanoString(timeString)
	assert.Equal(t, int64(1510860932), ts1.Seconds)
	assert.Equal(t, int32(12345678), ts1.Nanos)

	invalidTimeString1 := "0000-12-24T23:59:59.999999999Z"
	assert.Panics(t, func() { MustGetProtoTimestampFromRFC3339NanoString(invalidTimeString1) })

	invalidTimeString2 := "0000-12-2AT23:59:59.999999999Z"
	assert.Panics(t, func() { MustGetProtoTimestampFromRFC3339NanoString(invalidTimeString2) })
}

func TestRoundTimestamp(t *testing.T) {
	tsInvalid := &timestamppb.Timestamp{
		Seconds: -62235596800,
		Nanos:   123456789,
	}
	notRounded := RoundTimestamp(tsInvalid, time.Microsecond)
	assert.Equal(t, tsInvalid.AsTime(), notRounded.AsTime())

	ts1 := &timestamppb.Timestamp{
		Seconds: 1510860932,
		Nanos:   123456789,
	}
	rounded1 := RoundTimestamp(ts1, time.Microsecond)
	assert.Equal(t, ts1.Seconds, rounded1.Seconds)
	assert.Equal(t, int32(123457000), rounded1.Nanos)

	ts2 := &timestamppb.Timestamp{
		Seconds: 1510860932,
		Nanos:   987654321,
	}
	rounded2 := RoundTimestamp(ts2, time.Microsecond)
	assert.Equal(t, ts2.Seconds, rounded2.Seconds)
	assert.Equal(t, int32(987654000), rounded2.Nanos)

	ts3 := &timestamppb.Timestamp{
		Seconds: 1520860932,
		Nanos:   987654321,
	}
	rounded3 := RoundTimestamp(ts3, time.Millisecond)
	assert.Equal(t, ts3.Seconds, rounded3.Seconds)
	assert.Equal(t, int32(988000000), rounded3.Nanos)

	ts4 := &timestamppb.Timestamp{
		Seconds: 1510860932,
		Nanos:   123456789,
	}
	rounded4 := RoundTimestamp(ts4, time.Millisecond)
	assert.Equal(t, ts4.Seconds, rounded4.Seconds)
	assert.Equal(t, int32(123000000), rounded4.Nanos)
}
