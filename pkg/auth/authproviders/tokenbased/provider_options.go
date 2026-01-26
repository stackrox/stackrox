package tokenbased

import (
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
)

type TokenAuthProviderOption func(*tokenAuthProviderImpl)

// WithRevocationLayer registers a revocation layer in the token source,
// allowing it to revoke issued tokens.
func WithRevocationLayer(revocationLayer tokens.RevocationLayer) TokenAuthProviderOption {
	return func(ts *tokenAuthProviderImpl) {
		ts.revocationLayer = revocationLayer
	}
}

// WithRoleMapper registers a role mapper in the token source,
// allowing it to assign roles to token users.
func WithRoleMapper(roleMapper permissions.RoleMapper) TokenAuthProviderOption {
	return func(ts *tokenAuthProviderImpl) {
		ts.roleMapper = roleMapper
	}
}
