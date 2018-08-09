package user

import (
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/tokenbased"
)

// Identity is a token-based identity that represents a user
// from an auth provider.
// This is for humans (not machines) who use the system.
type Identity interface {
	tokenbased.Identity
	// AuthProvider is the AuthProvider that this
	// user identity was derived from.
	AuthProvider() authproviders.AuthProvider
}

type identityImpl struct {
	tokenbased.Identity
	authProvider authproviders.AuthProvider
}

func (i *identityImpl) AuthProvider() authproviders.AuthProvider {
	return i.authProvider
}

// NewIdentity returns a user identity with the provided parameters.
func NewIdentity(tbIdentity tokenbased.Identity, authProvider authproviders.AuthProvider) Identity {
	return &identityImpl{
		Identity:     tbIdentity,
		authProvider: authProvider,
	}
}
