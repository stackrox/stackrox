package centralproxy

import (
	"context"

	"github.com/stackrox/rox/pkg/centralproxy"
)

type proxyPermissionSetKey struct{}

var proxyPathPermissionSets = map[string]string{
	"/metrics": centralproxy.MetricsPermissionSetID,
}

func withMetricsScrape(ctx context.Context) context.Context {
	return context.WithValue(ctx, proxyPermissionSetKey{}, proxyPathPermissionSets["/metrics"])
}

func proxyPermissionSetFromContext(ctx context.Context) string {
	permissionSetID, _ := ctx.Value(proxyPermissionSetKey{}).(string)
	return permissionSetID
}
