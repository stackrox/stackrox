package authn

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
)

// Identity represents the identity of an entity accessing a service.
type Identity interface {
	UID() string
	FriendlyName() string

	Role() *storage.Role
	Service() *storage.ServiceIdentity

	Expiry() time.Time
	ExternalAuthProvider() authproviders.Provider
}
