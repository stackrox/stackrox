package proto

import "google.golang.org/protobuf/proto"

// Equal checks if the two proto messages are equal.
// TODO: use vtprotobuf optimized equality checking.
func Equal(m1 proto.Message, m2 proto.Message) bool {
	return proto.Equal(m1, m2)
}
