package protoutils

import (
	"github.com/gogo/protobuf/types"
	golangProto "github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/secrets"
)

// MarshalAny correctly marshals a proto message into an Any
// which is required because of our use of gogo and golang proto
// TODO(cgorman) Resolve this by correctly implementing the other proto
// pieces
func MarshalAny(msg protocompat.Message) (*types.Any, error) {
	a, err := types.MarshalAny(msg)
	if err != nil {
		return nil, err
	}
	a.TypeUrl = golangProto.MessageName(msg)
	return a, nil
}

// RequestToAny converts an input protobuf message to the generic protobuf Any type,
// with the secrets scrubbed.
func RequestToAny(req interface{}) *types.Any {
	if req == nil {
		return nil
	}
	msg, ok := req.(protocompat.Message)
	if !ok {
		return nil
	}

	// Must clone before potentially modifying it
	msg = protocompat.Clone(msg)
	secrets.ScrubSecretsFromStructWithReplacement(msg, "")
	a, err := MarshalAny(msg)
	if err != nil {
		return nil
	}
	return a
}
