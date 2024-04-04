package protoconv

import (
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
)

func TestConvertTimeString(t *testing.T) {
	cases := []struct {
		input  string
		output *types.Timestamp
	}{
		{
			input:  "",
			output: nil,
		},
		{
			input:  "malformed",
			output: nil,
		},
		{
			input:  "2018-02-07T23:29Z",
			output: protocompat.GetProtoTimestampFromSeconds(1518046140),
		},
		{
			input:  "2019-01-20T00:00:00Z",
			output: protocompat.GetProtoTimestampFromSeconds(1547942400),
		},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			assert.Equal(t, c.output, ConvertTimeString(c.input))
		})
	}
}

func TestReadableTime(t *testing.T) {
	ts1 := protocompat.GetProtoTimestampFromSeconds(1547942400)
	readable1 := ReadableTime(ts1)
	assert.Equal(t, "2019-01-20 00:00:00", readable1)

	invalidTs := protocompat.GetProtoTimestampFromSeconds(-62234567890)
	unreadable := ReadableTime(invalidTs)
	assert.Equal(t, "<malformed time>", unreadable)
}

func TestTimeBeforeDays(t *testing.T) {
	now := protocompat.TimestampNow()

	threeDaysAgo := TimeBeforeDays(3)

	nowMinusThreeDaysSeconds := now.Seconds - int64(3*24*3600)
	deltaSeconds := threeDaysAgo.Seconds - nowMinusThreeDaysSeconds
	assert.True(t, deltaSeconds >= 0)
	assert.True(t, deltaSeconds < 3)
}
