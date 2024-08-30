package protocompat

import (
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// Message is implemented by generated protocol buffer messages.
type Message = proto.Message

// Clone returns a deep copy of a protocol buffer.
// Deprecated: Use CloneVT or CloneMessageVT instead.
func Clone(msg proto.Message) proto.Message {
	if vtMsg, ok := msg.(interface{ CloneMessageVT() proto.Message }); ok {
		return vtMsg.CloneMessageVT()
	}
	return proto.Clone(msg)
}

// MarshalTextString writes a given protocol buffer in text format,
// returning the string directly.
func MarshalTextString(m proto.Message) string {
	return prototext.MarshalOptions{Multiline: true}.Format(m)
}

// Merge merges src into dst.
// Required and optional fields that are set in src will be set to that value in dst.
// Elements of repeated fields will be appended.
// Merge panics if src and dst are not the same type, or if dst is nil.
func Merge(dst, src Message) {
	proto.Merge(dst, src)
}
