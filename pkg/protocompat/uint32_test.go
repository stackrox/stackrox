package protocompat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestProtoUInt32Value(t *testing.T) {
	input1 := uint32(0)
	expectedVal1 := &wrapperspb.UInt32Value{
		Value: input1,
	}

	val1 := ProtoUInt32Value(input1)
	assert.Equal(t, expectedVal1.Value, val1.Value)

	input2 := uint32(1234567890)
	expectedVal2 := &wrapperspb.UInt32Value{
		Value: input2,
	}

	val2 := ProtoUInt32Value(input2)
	assert.Equal(t, expectedVal2.Value, val2.Value)
}
