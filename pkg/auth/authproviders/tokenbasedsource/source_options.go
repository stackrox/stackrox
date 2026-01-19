package tokenbasedsource

import (
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
)

type TokenSourceOption func(*tokenSourceImpl)

// WithRevocationLayer registers a revocation layer in the token source,
// allowing it to revoke issued tokens.
func WithRevocationLayer(revocationLayer tokens.RevocationLayer) TokenSourceOption {
	return func(ts *tokenSourceImpl) {
		ts.revocationLayer = revocationLayer
	}
}

// WithRoleMapper registers a role mapper in the token source,
// allowing it to assign roles to token users.
func WithRoleMapper(roleMapper permissions.RoleMapper) TokenSourceOption {
	return func(ts *tokenSourceImpl) {
		ts.roleMapper = roleMapper
	}
}
