package types

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// Any is an alias for the anypb.Any type.
type Any = anypb.Any

// MarshalAny marshals the given protobuf message into an Any protobuf object.
func MarshalAny(msg proto.Message) (*Any, error) {
	res := new(Any)
	return res, anypb.MarshalFrom(res, msg, proto.MarshalOptions{})
}
