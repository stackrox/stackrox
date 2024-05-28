package protocompat

import (
	"io"

	"github.com/gogo/protobuf/proto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/protoadapt"
)

// Message is implemented by generated protocol buffer messages.
type Message = proto.Message

// Clone returns a deep copy of a protocol buffer.
func Clone(msg proto.Message) proto.Message {
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

// Marshal takes a protocol buffer message and encodes it into
// the wire format, returning the data. This is the main entry point.
func Marshal(msg proto.Message) ([]byte, error) {
	return proto.Marshal(msg)
}

// MarshalTextString writes a given protocol buffer in text format,
// returning the string directly..
func MarshalTextString(msg proto.Message) string {
	return proto.MarshalTextString(msg)
}

// MarshalToProtoJSONBytes writes a given protocol buffer in JSON format,
// returning the data as byte array.
func MarshalToProtoJSONBytes(msg proto.Message) ([]byte, error) {
	msg2 := protoadapt.MessageV2Of(msg)
	m := protojson.MarshalOptions{}
	return m.Marshal(msg2)
}

// MarshalToIndentedProtoJSONBytes writes a given protocol buffer in JSON format,
// returning the data as byte array.
func MarshalToIndentedProtoJSONBytes(msg proto.Message) ([]byte, error) {
	msg2 := protoadapt.MessageV2Of(msg)
	m := protojson.MarshalOptions{
		Indent: "  ",
	}
	return m.Marshal(msg2)
}

// MarshalToProtoJSONString writes a given protocol buffer in JSON format,
// returning the data as a string.
func MarshalToProtoJSONString(msg proto.Message) (string, error) {
	jsonBytes, err := MarshalToProtoJSONBytes(msg)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// MarshalToIndentedProtoJSONString writes a given protocol buffer in JSON format,
// returning the data as a string.
func MarshalToIndentedProtoJSONString(msg proto.Message) (string, error) {
	jsonBytes, err := MarshalToIndentedProtoJSONBytes(msg)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// Unmarshal parses the protocol buffer representation in buf and places
// the decoded result in pb. If the struct underlying pb does not match
// the data in buf, the results can be unpredictable.
//
// Unmarshal resets pb before starting to unmarshal, so any existing data
// in pb is always removed.
func Unmarshal(dAtA []byte, msg proto.Message) error {
	return proto.Unmarshal(dAtA, msg)
}

// UnmarshalProtoJSON parses the json representation in reader and places
// the decoded result in msg.
func UnmarshalProtoJSON(reader io.Reader, msg proto.Message) error {
	x, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	unmarshaler := protojson.UnmarshalOptions{}
	msg2 := protoadapt.MessageV2Of(msg)
	unmarshaler.Unmarshal(x, msg2)
	msg = protoadapt.MessageV1Of(msg2)
	return nil
}

// Unmarshaler is a generic interface type wrapping around types that implement protobuf Unmarshaler.
type Unmarshaler[T any] interface {
	proto.Unmarshaler
	*T
}

// ClonedUnmarshaler is a generic interface type wrapping around types that implement protobuf Unmarshaler
// and that have a Clone deep-copy method.
type ClonedUnmarshaler[T any] interface {
	Clone() *T
	proto.Unmarshaler
	*T
}

// Merge merges src into dst.
// Required and optional fields that are set in src will be set to that value in dst.
// Elements of repeated fields will be appended.
// Merge panics if src and dst are not the same type, or if dst is nil.
func Merge(dst, src Message) {
	proto.Merge(dst, src)
}
