package protoconv

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestConvertTimeString(t *testing.T) {
	cases := []struct {
		input  string
		output *timestamppb.Timestamp
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
			assert.Equal(t, c.output.AsTime(), ConvertTimeString(c.input).AsTime())
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

func TestConvertMicroTSToProtobufTS(t *testing.T) {
	time0 := timestamp.MicroTS(0)
	timestamp0 := ConvertMicroTSToProtobufTS(time0)
	assert.NotNil(t, timestamp0)
	assert.Equal(t, int64(0), timestamp0.Seconds)
	assert.Equal(t, int32(0), timestamp0.Nanos)

	timestamp1 := &timestamppb.Timestamp{
		Seconds: 1518046140,
		Nanos:   123456789,
	}
	time1 := timestamp.FromProtobuf(timestamp1)
	convertedTimestamp1 := ConvertMicroTSToProtobufTS(time1)
	assert.NotNil(t, convertedTimestamp1)
	assert.Equal(t, timestamp1.Seconds, convertedTimestamp1.Seconds)
	assert.Equal(t, int32(123456000), convertedTimestamp1.Nanos)
}

func TestConvertTimestampToTimeOrDefault(t *testing.T) {
	defaultGoTime := time.Unix(0x7E57ED, 0xAC5)
	protoTime := &timestamppb.Timestamp{
		Seconds: 1518046140,
		Nanos:   123456789,
	}
	goTime := ConvertTimestampToTimeOrDefault(protoTime, defaultGoTime)
	assert.Equal(t, goTime.Unix(), protoTime.GetSeconds())
	assert.Equal(t, goTime.Nanosecond(), int(protoTime.GetNanos()))

	goTime = ConvertTimestampToTimeOrDefault(nil, defaultGoTime)
	assert.Equal(t, goTime.UTC(), defaultGoTime.UTC())

	// Negative Nanos is not valid for Timestamp.
	invalidProtoTime := &timestamppb.Timestamp{
		Nanos: -1,
	}
	goTime = ConvertTimestampToTimeOrDefault(invalidProtoTime, defaultGoTime)
	assert.Equal(t, goTime.UTC(), defaultGoTime.UTC())
}

func TestConvertTimeToTimestamp(t *testing.T) {
	goTime := time.Unix(0x7E57ED, 0xAC5)
	protoTime := ConvertTimeToTimestamp(goTime)
	assert.Equal(t, goTime.Unix(), protoTime.GetSeconds())
	assert.Equal(t, goTime.Nanosecond(), int(protoTime.GetNanos()))

	goTimeBeforeCall := time.Now()
	invalidGoTime := goTime.AddDate(-10000, 0, 0)
	protoTime = ConvertTimeToTimestamp(invalidGoTime)

	// When conversion fails, `ConvertTimeToTimestamp` will return the current time.
	// We are checking that the returned time is after the time fetched before the call.
	assert.GreaterOrEqual(t, protoTime.GetSeconds()*time.Second.Nanoseconds()+int64(protoTime.GetNanos())*time.Nanosecond.Nanoseconds(), goTimeBeforeCall.UnixNano())
}

func TestConvertTimeToTimestampOrNil(t *testing.T) {
	goTime := time.Unix(0x7E57ED, 0xAC5)
	protoTime := ConvertTimeToTimestampOrNil(goTime)
	assert.Equal(t, goTime.Unix(), protoTime.GetSeconds())
	assert.Equal(t, goTime.Nanosecond(), int(protoTime.GetNanos()))

	invalidGoTime := goTime.AddDate(-10000, 0, 0)
	invalidProtoTime := ConvertTimeToTimestampOrNil(invalidGoTime)
	assert.Nil(t, invalidProtoTime)
}

func TestMustConvertTimeToTimestamp(t *testing.T) {
	goTime := time.Unix(0x7E57ED, 0xAC5)
	protoTime := MustConvertTimeToTimestamp(goTime)
	assert.Equal(t, goTime.Unix(), protoTime.GetSeconds())
	assert.Equal(t, goTime.Nanosecond(), int(protoTime.GetNanos()))

	invalidGoTime := goTime.AddDate(-10000, 0, 0)
	assert.Panics(t, func() {
		MustConvertTimeToTimestamp(invalidGoTime)
	})
}
