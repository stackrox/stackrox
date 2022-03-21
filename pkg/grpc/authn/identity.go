package authn

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

//go:generate mockgen-wrapper

// Identity represents the identity of an entity accessing a service.
type Identity interface {
	UID() string
	FriendlyName() string
	// FullName could be empty
	FullName() string

	User() *storage.UserInfo
	Permissions() map[string]storage.Access
	Roles() []permissions.ResolvedRole

	Service() *storage.ServiceIdentity
	Attributes() map[string][]string
	// ValidityPeriod returns the range (begin, end) in which the identity
	// remains valid.
	ValidityPeriod() (time.Time, time.Time)

	ExternalAuthProvider() authproviders.Provider
}
