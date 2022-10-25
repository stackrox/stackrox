package protoutils

import (
	proto2 "github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/pkg/transitional/protocompat/proto"
	golangProto "github.com/stackrox/rox/pkg/transitional/protocompat/proto"
	"github.com/stackrox/rox/pkg/transitional/protocompat/types"
)

// MarshalAny correctly marshals a proto message into an Any
// which is required because of our use of gogo and golang proto
// TODO(cgorman) Resolve this by correctly implementing the other proto
// pieces
func MarshalAny(msg proto.Message) (*types.Any, error) {
	any, err := types.MarshalAny(msg)
	if err != nil {
		return nil, err
	}
	any.TypeUrl = golangProto.MessageName(proto2.MessageV1(msg))
	return any, nil
}
