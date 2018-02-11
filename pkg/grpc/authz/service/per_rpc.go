package service

import (
	"context"

	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz"
)

// PerRPC allows the application of a default policy, with exceptions
// for individual RPCs.
type PerRPC struct {
	// Default is applied if no specific authorization is listed in Authorizers.
	Default authz.Authorizer
	// Authorizers maps full RPC method names (e.g., /v1.MyGRPCService/MyMethod)
	// to the Authorizer that should be used for them.
	Authorizers map[string]authz.Authorizer
}

// Authorized applies the default or per-RPC authorizer, as appropriate.
func (pr PerRPC) Authorized(ctx context.Context, rpcName string) error {
	if a := pr.Authorizers[rpcName]; a != nil {
		return a.Authorized(ctx)
	}
	return pr.Default.Authorized(ctx)
}
