package allow

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc/authz"
)

type anonymous struct{}

func (anonymous) Authorized(context.Context, string) error { return nil }

// Anonymous returns an Authorizer that allows all access.
func Anonymous() authz.Authorizer { return anonymous{} }
