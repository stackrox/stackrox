package centralproxy

import (
	"context"

	apiV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/centralproxy"
)

type proxyPermissionSetKey struct{}
type proxyTokenProfileKey struct{}

// proxyTokenProfile is selected only after Sensor authenticates a caller and
// checks the incoming method and path against its local client policy.
type proxyTokenProfile struct {
	identity        string
	permissionSetID string
	httpScopes      []*apiV1.HTTPRequestScope
	clusterScope    string
	legacyConsole   bool
}

func withProxyTokenProfile(ctx context.Context, profile *proxyTokenProfile) context.Context {
	return context.WithValue(ctx, proxyTokenProfileKey{}, profile)
}

func proxyTokenProfileFromContext(ctx context.Context) *proxyTokenProfile {
	profile, _ := ctx.Value(proxyTokenProfileKey{}).(*proxyTokenProfile)
	return profile
}

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
