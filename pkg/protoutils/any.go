package protoutils

import (
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	golangProto "github.com/golang/protobuf/proto"
)

// MarshalAny correctly marshals a proto message into an Any
// which is required because of our use of gogo and golang proto
// TODO(cgorman) Resolve this by correctly implementing the other proto
// pieces
func MarshalAny(msg proto.Message) (*types.Any, error) {
	a, err := types.MarshalAny(msg)
	if err != nil {
		return nil, err
	}
	a.TypeUrl = golangProto.MessageName(msg)
	return a, nil
}
