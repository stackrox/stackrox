package protocompat

import (
	"github.com/gogo/protobuf/types"
)

// ProtoUInt32Value builds a *types.UInt32Value with the input value as `Value` field
func ProtoUInt32Value(val uint32) *types.UInt32Value {
	return &types.UInt32Value{Value: val}
}
