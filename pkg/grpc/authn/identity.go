package authn

import (
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/authproviders"
)

// Identity represents the identity of an entity accessing a service.
type Identity interface {
	UID() string
	FriendlyName() string

	Role() *v1.Role
	Service() *v1.ServiceIdentity

	Expiry() time.Time
	ExternalAuthProvider() authproviders.AuthProvider
}
