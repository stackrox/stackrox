package basic

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/timeutil"
)

type identity struct {
	username string
	role     *storage.Role
}

func (i identity) UID() string {
	return i.username
}

func (i identity) FriendlyName() string {
	return i.username
}

func (i identity) Role() *storage.Role {
	return i.role
}

func (i identity) Service() *storage.ServiceIdentity {
	return nil
}

func (i identity) Expiry() time.Time {
	return timeutil.Max
}

func (i identity) ExternalAuthProvider() authproviders.AuthProvider {
	return nil
}

func (i identity) isBasicAuthIdentity() {}

func (i identity) AsExternalUser() *tokens.ExternalUserClaim {
	return &tokens.ExternalUserClaim{
		UserID: i.username,
	}
}

// Identity is an extension of the identity interface for user authenticating via Basic authentication.
type Identity interface {
	authn.Identity

	// AsExternalUser returns the claims
	AsExternalUser() *tokens.ExternalUserClaim

	// isBasicAuthIdentity is a marker method.
	isBasicAuthIdentity()
}
