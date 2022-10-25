package timestamp

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stackrox/rox/pkg/transitional/protocompat/types"
	"github.com/stretchr/testify/assert"
)

func TestGoTimeToMicroTS(t *testing.T) {
	goTime := time.Unix(500, 5000)
	microTS := FromGoTime(goTime)
	assert.Equal(t, microTS, MicroTS(500000005))
	assert.Equal(t, goTime, microTS.GoTime())
}

func TestNilGoogleProtobufIsZero(t *testing.T) {
	var ts *timestamp.Timestamp
	assert.Zero(t, FromProtobuf(ts))
}

func TestNilGogoProtobufIsZero(t *testing.T) {
	var ts *types.Timestamp
	assert.Zero(t, FromProtobuf(ts))
}

func TestElapsedSince(t *testing.T) {
	ts1 := MicroTS(1000000)
	ts2 := MicroTS(20000000)
	assert.Equal(t, 19*time.Second, ts2.ElapsedSince(ts1))
	assert.True(t, ts2.After(ts1))
	assert.False(t, ts1.After(ts2))
}
