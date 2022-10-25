package proto

import (
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// MarshalTextString marshals the given message in prototext format.
func MarshalTextString(m proto.Message) string {
	return prototext.Format(m)
}
