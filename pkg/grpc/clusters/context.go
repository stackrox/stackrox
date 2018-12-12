package clusters

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
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

	if svc.GetType() != storage.ServiceType_SENSOR_SERVICE {
		return ""
	}

	return svc.GetId()
}
