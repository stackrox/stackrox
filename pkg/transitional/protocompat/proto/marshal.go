package proto

import (
	"github.com/stackrox/rox/pkg/reflectutils"
	"google.golang.org/protobuf/proto"
)

// Marshal marshals the given message, using optimized vtprotobuf marshaling if available.
func Marshal(m proto.Message) ([]byte, error) {
	if reflectutils.IsNil(m) {
		return nil, ErrNil
	}
	if marshalVT, ok := m.(interface{ MarshalVT() ([]byte, error) }); ok {
		return marshalVT.MarshalVT()
	}
	return proto.Marshal(m)
}
