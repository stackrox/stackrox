package proto

import "google.golang.org/protobuf/proto"

// Unmarshal unmarshals the given bytes into a protobuf message, using optimized vtprotobuf unmarshaling if available.
func Unmarshal(data []byte, m proto.Message) error {
	if unmarshalVT, ok := m.(interface{ UnmarshalVT([]byte) error }); ok {
		return unmarshalVT.UnmarshalVT(data)
	}
	return proto.Unmarshal(data, m)
}
