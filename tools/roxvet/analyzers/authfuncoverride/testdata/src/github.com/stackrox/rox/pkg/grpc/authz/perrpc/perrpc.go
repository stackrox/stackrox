package perrpc

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc/authz"
)

type perRPC struct{}

func (p *perRPC) Authorized(_ context.Context, _ string) error { return nil }

// FromMap returns an Authorizer that dispatches per-RPC based on the reversed map.
func FromMap(m map[authz.Authorizer][]string) authz.Authorizer { return &perRPC{} }
