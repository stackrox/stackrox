package centralproxy

import (
	"context"

	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
)

type proxyTokenProfileKey struct{}

// proxyTokenProfile is assigned after client authentication and local route checks.
// identity is an audit attribute, not proof of possession of the original credential.
type proxyTokenProfile struct {
	identity            string
	permissionSets      []*centralv1.SelectedPermissionSet
	singlePermissionSet bool
	clusterScope        string
	legacy              bool
}

func withProxyTokenProfile(ctx context.Context, profile *proxyTokenProfile) context.Context {
	return context.WithValue(ctx, proxyTokenProfileKey{}, profile)
}

func proxyTokenProfileFromContext(ctx context.Context) *proxyTokenProfile {
	profile, _ := ctx.Value(proxyTokenProfileKey{}).(*proxyTokenProfile)
	return profile
}
