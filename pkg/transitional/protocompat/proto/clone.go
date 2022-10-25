package proto

import "google.golang.org/protobuf/proto"

// Clone clones a given message, using optimized vtprotobuf cloning if available.
func Clone(m proto.Message) proto.Message {
	if cloneVT, ok := m.(interface{ CloneGenericVT() proto.Message }); ok {
		return cloneVT.CloneGenericVT()
	}
	return proto.Clone(m)
}
