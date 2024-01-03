package protocompat

import (
	"github.com/gogo/protobuf/proto"
)

// Equal returns true if both protobuf messages are equal
func Equal(a proto.Message, b proto.Message) bool {
	return proto.Equal(a, b)
}
