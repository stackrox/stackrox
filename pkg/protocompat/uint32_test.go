package protocompat

import (
	"testing"

	"github.com/stackrox/rox/pkg/protoassert"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestProtoUInt32Value(t *testing.T) {
	input1 := uint32(0)
	expectedVal1 := &wrapperspb.UInt32Value{
		Value: input1,
	}

	val1 := ProtoUInt32Value(input1)
	protoassert.Equal(t, expectedVal1, val1)

	input2 := uint32(1234567890)
	expectedVal2 := &wrapperspb.UInt32Value{
		Value: input2,
	}

	val2 := ProtoUInt32Value(input2)
	protoassert.Equal(t, expectedVal2, val2)
}
