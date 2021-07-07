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
