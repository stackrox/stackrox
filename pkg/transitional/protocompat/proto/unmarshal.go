package proto

import (
	"github.com/stackrox/rox/pkg/reflectutils"
	"google.golang.org/protobuf/proto"
)

// Unmarshal unmarshals the given bytes into a protobuf message, using optimized vtprotobuf unmarshaling if available.
func Unmarshal(data []byte, m proto.Message) error {
	if reflectutils.IsNil(m) {
		return ErrNil
	}
	if unmarshalVT, ok := m.(interface{ UnmarshalVT([]byte) error }); ok {
		return unmarshalVT.UnmarshalVT(data)
	}
	return proto.Unmarshal(data, m)
}
