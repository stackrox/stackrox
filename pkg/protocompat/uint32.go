package protocompat

import (
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// ProtoUInt32Value builds a *types.UInt32Value with the input value as `Value` field
func ProtoUInt32Value(val uint32) *wrapperspb.UInt32Value {
	return &wrapperspb.UInt32Value{Value: val}
}
