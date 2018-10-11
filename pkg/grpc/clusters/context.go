package clusters

import (
	"context"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authn"
)

// IDFromContext retrieves the cluster ID from the given context.
func IDFromContext(ctx context.Context) string {
	id := authn.IdentityFromContext(ctx)
	if id == nil {
		return ""
	}
	svc := id.Service()
	if svc == nil {
		return ""
	}

	if svc.GetType() != v1.ServiceType_SENSOR_SERVICE {
		return ""
	}

	return svc.GetId()
}
