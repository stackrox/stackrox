package deny

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc/authz"
)

type everyone struct{}

func (everyone) Authorized(context.Context, string) error { return nil }

// Everyone returns an Authorizer that denies all access.
func Everyone() authz.Authorizer { return everyone{} }
