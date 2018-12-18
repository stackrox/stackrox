package authproviders

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
)

// An Provider is an authenticator which is based on an external service, like auth0.
type Provider interface {
	tokens.Source

	Name() string
	Type() string
	Enabled() bool
	Backend() Backend
	RoleMapper() permissions.RoleMapper

	// StorageView returns a description of the authentication provider in protobuf format.
	StorageView() *storage.AuthProvider

	// RecordSuccess should be called the first time a user successfully logs in through an auth provider, to mark it as
	// validated. This is used to prevent a user from accidentally locking themselves out of the system by setting up a
	// misconfigured auth provider.
	RecordSuccess() error
}
