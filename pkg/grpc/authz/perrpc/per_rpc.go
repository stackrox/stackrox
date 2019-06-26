package perrpc

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/deny"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// perRPC allows the application of a default policy, with exceptions
// for individual RPCs.
type perRPC struct {
	// Default is applied if no specific authorization is listed in Authorizers.
	Default authz.Authorizer

	// Authorizers maps full RPC method names (e.g., /v1.MyGRPCService/MyMethod)
	// to the Authorizer that should be used for them.
	Authorizers map[string]authz.Authorizer
}

// Authorized applies the default or per-RPC authorizer, as appropriate.
func (pr *perRPC) Authorized(ctx context.Context, rpcName string) error {
	if a, exists := pr.Authorizers[rpcName]; exists {
		return a.Authorized(ctx, rpcName)
	}
	return pr.Default.Authorized(ctx, rpcName)
}

// FromMap returns a perRPC authorizer from a reversed map,
// for clients which find it more convenient this way.
// It sets the default to Deny, to force programmers to specify all RPC methods in the map.
// BE CAREFUL WHEN USING THIS FUNCTION: if any of your authorizers is not hashable, it will
// cause a runtime panic. If you want to use a struct that contains a slice/map as an Authorizer,
// make sure that it is a pointer to your struct that implements Authorizer.
// (An example is the perRPC struct in this package.)
func FromMap(m map[authz.Authorizer][]string) authz.Authorizer {
	authorizers := make(map[string]authz.Authorizer)
	for authorizer, rpcNames := range m {
		for _, rpcName := range rpcNames {
			// This is a programming error, and will be rewarded with a deny.Everyone() to force
			// the programmer to right their ways.
			if _, exists := authorizers[rpcName]; exists {
				log.Errorf("rpcName %s mapped to multiple authorizers in map: %#v", rpcName, m)
				return deny.Everyone()
			}
			authorizers[rpcName] = authorizer
		}
	}
	return &perRPC{
		Default:     deny.Everyone(),
		Authorizers: authorizers,
	}
}
