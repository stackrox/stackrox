package protocompat

import (
	"encoding/json"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
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

// ErrNil is the error returned if Marshal is called with nil.
var ErrNil = errors.New("proto: Marshal called with nil")

// Marshal takes a protocol buffer message and encodes it into
// the wire format, returning the data. This is the main entry point.
func Marshal[T any, PT marshalable[T]](msg PT) ([]byte, error) {
	var null PT
	if msg == null {
		return nil, ErrNil
	}
	return msg.MarshalVT()
}

type marshalable[T any] interface {
	*T
	MarshalVT() ([]byte, error)
}

// MarshalTextString writes a given protocol buffer in text format,
// returning the string directly.
func MarshalTextString(m proto.Message) string {
	return prototext.MarshalOptions{Multiline: true}.Format(m)
}

// MarshalMap marshals a proto message to a map[string]interface{} type.
func MarshalMap(m proto.Message) (map[string]interface{}, error) {
	marshalledProto, err := protojson.Marshal(m)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert proto msg to json")
	}

	dest := map[string]interface{}{}
	err = json.Unmarshal(marshalledProto, &dest)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert to an unstructured map")
	}

	return dest, nil
}

// Unmarshaler is a generic interface type wrapping around types that implement protobuf Unmarshaler.
type Unmarshaler[T any] interface {
	UnmarshalVT(dAtA []byte) error
	*T
}

// ClonedUnmarshaler is a generic interface type wrapping around types that implement protobuf Unmarshaler
// and that have a Clone deep-copy method.
type ClonedUnmarshaler[T any] interface {
	Unmarshaler[T]
	CloneVT() *T
}

// Merge merges src into dst.
// Required and optional fields that are set in src will be set to that value in dst.
// Elements of repeated fields will be appended.
// Merge panics if src and dst are not the same type, or if dst is nil.
func Merge(dst, src Message) {
	proto.Merge(dst, src)
}
