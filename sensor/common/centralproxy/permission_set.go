package centralproxy

import (
	"context"

	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
)

type proxyTokenProfileKey struct{}

// proxyTokenProfile is selected only after Sensor authenticates a caller and
// checks the incoming method and path against its local client policy.
type proxyTokenProfile struct {
	identity       string
	permissionSets []*centralv1.ScopedPermissionSet
	clusterScope   string
	legacyConsole  bool
}

func withProxyTokenProfile(ctx context.Context, profile *proxyTokenProfile) context.Context {
	return context.WithValue(ctx, proxyTokenProfileKey{}, profile)
}

func proxyTokenProfileFromContext(ctx context.Context) *proxyTokenProfile {
	profile, _ := ctx.Value(proxyTokenProfileKey{}).(*proxyTokenProfile)
	return profile
}
