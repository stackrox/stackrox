package protocompat

import (
	"github.com/gogo/protobuf/proto"
)

// Equal returns true iff protocol buffers a and b are equal. The arguments must both be pointers to protocol buffer structs.
//
// Equality is defined in this way:
//
// * Two messages are equal iff they are the same type, corresponding fields are equal, unknown field sets are equal, and extensions sets are equal.
// * Two set scalar fields are equal iff their values are equal. If the fields are of a floating-point type, remember that NaN != x for all x, including NaN. If the message is defined in a proto3 .proto file, fields are not "set"; specifically, zero length proto3 "bytes" fields are equal (nil == {}).
// * Two repeated fields are equal iff their lengths are the same, and their corresponding elements are equal. Note a "bytes" field, although represented by []byte, is not a repeated field and the rule for the scalar fields described above applies.
// * Two unset fields are equal.
// * Two unknown field sets are equal if their current encoded state is equal.
// * Two extension sets are equal iff they have corresponding elements that are pairwise equal.
// * Two map fields are equal iff their lengths are the same, and they contain the same set of elements. Zero-length map fields are equal.
// * Every other combination of things are not equal.
//
// The return value is undefined if a and b are not protocol buffers.
func Equal(a proto.Message, b proto.Message) bool {
	return proto.Equal(a, b)
}

// MarshalTextString writes a given protocol buffer in text format,
// returning the string directly..
func MarshalTextString(msg proto.Message) string {
	return proto.MarshalTextString(msg)
}
