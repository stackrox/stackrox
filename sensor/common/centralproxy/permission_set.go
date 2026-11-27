package centralproxy

import "context"

type proxyTokenProfileKey struct{}

// proxyTokenProfile is assigned after client authentication and local route checks.
// identity is an audit attribute, not proof of possession of the original credential.
type proxyTokenProfile struct {
	identity          string
	permissionSetID   string
	permissionSetName string
	clusterScope      string
	legacyConsole     bool
}

func withProxyTokenProfile(ctx context.Context, profile *proxyTokenProfile) context.Context {
	return context.WithValue(ctx, proxyTokenProfileKey{}, profile)
}

func proxyTokenProfileFromContext(ctx context.Context) *proxyTokenProfile {
	profile, _ := ctx.Value(proxyTokenProfileKey{}).(*proxyTokenProfile)
	return profile
}
