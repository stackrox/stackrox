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
