package protocompat

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/secrets"
)

// RequestToAny converts an input protobuf message to the generic protobuf Any type,
// with the secrets scrubbed.
func RequestToAny(req interface{}) *types.Any {
	if req == nil {
		return nil
	}
	msg, ok := req.(Message)
	if !ok {
		return nil
	}

	// Must clone before potentially modifying it
	msg = Clone(msg)
	secrets.ScrubSecretsFromStructWithReplacement(msg, "")
	a, err := protoutils.MarshalAny(msg)
	if err != nil {
		return nil
	}
	return a
}
