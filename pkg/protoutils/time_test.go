package protoutils

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/assert"
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
