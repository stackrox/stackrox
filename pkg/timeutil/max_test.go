package timeutil

import (
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
)

func TestMaxProtoValid(t *testing.T) {
	t.Parallel()

	tsProto, err := protocompat.ConvertTimeToTimestampOrError(MaxProtoValid)
	assert.NoError(t, err)

	ts, err := types.TimestampFromProto(tsProto)
	assert.NoError(t, err)
	assert.Equal(t, MaxProtoValid, ts)
}
