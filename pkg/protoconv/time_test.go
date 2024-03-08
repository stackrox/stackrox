package protoconv

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
)

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
