package protocompat

import (
	"errors"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// Message is implemented by generated protocol buffer messages.
type Message = proto.Message

// Clone returns a deep copy of a protocol buffer.
func Clone(msg proto.Message) proto.Message {
	if vtMsg, ok := msg.(interface{ CloneMessageVT() proto.Message }); ok {
		return vtMsg.CloneMessageVT()
	}
	return proto.Clone(msg)
}

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

// ErrNil is the error returned if Marshal is called with nil.
var ErrNil = errors.New("proto: Marshal called with nil")

// Marshal takes a protocol buffer message and encodes it into
// the wire format, returning the data. This is the main entry point.
func Marshal[T any, PT marshalable[T]](msg PT) ([]byte, error) {
	var null PT
	if msg == null {
		return nil, ErrNil
	}
	return msg.Marshal()
}

type marshalable[T any] interface {
	*T
	Marshal() ([]byte, error)
}

// MarshalTextString writes a given protocol buffer in text format,
// returning the string directly.
func MarshalTextString(m proto.Message) string {
	return prototext.MarshalOptions{Multiline: true}.Format(m)
}

// Unmarshal parses the protocol buffer representation in buf and places
// the decoded result in pb. If the struct underlying pb does not match
// the data in buf, the results can be unpredictable.
//
// Unmarshal resets pb before starting to unmarshal, so any existing data
// in pb is always removed.
func Unmarshal[T any, PT Unmarshaler[T]](dAtA []byte, msg PT) error {
	if dAtA == nil {
		return ErrNil
	}

	return msg.Unmarshal(dAtA)
}

// Unmarshaler is a generic interface type wrapping around types that implement protobuf Unmarshaler.
type Unmarshaler[T any] interface {
	Unmarshal(dAtA []byte) error
	*T
}

// ClonedUnmarshaler is a generic interface type wrapping around types that implement protobuf Unmarshaler
// and that have a Clone deep-copy method.
type ClonedUnmarshaler[T any] interface {
	Clone() *T
	Unmarshal(dAtA []byte) error
	*T
}

// Merge merges src into dst.
// Required and optional fields that are set in src will be set to that value in dst.
// Elements of repeated fields will be appended.
// Merge panics if src and dst are not the same type, or if dst is nil.
func Merge(dst, src Message) {
	proto.Merge(dst, src)
}
