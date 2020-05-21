package tokenbased

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
)

type roleBasedIdentity struct {
	uid          string
	username     string
	friendlyName string
	fullName     string
	perms        *storage.Role
	roles        []*storage.Role
	expiry       time.Time
	attributes   map[string][]string
	authProvider authproviders.Provider
}

func (i *roleBasedIdentity) UID() string {
	return i.uid
}

func (i *roleBasedIdentity) FriendlyName() string {
	return i.friendlyName
}

func (i *roleBasedIdentity) FullName() string {
	return i.fullName
}

func (i *roleBasedIdentity) Permissions() *storage.Role {
	return i.perms
}

func (i *roleBasedIdentity) Roles() []*storage.Role {
	return i.roles
}

func (i *roleBasedIdentity) Service() *storage.ServiceIdentity {
	return nil
}

func (i *roleBasedIdentity) User() *storage.UserInfo {
	return &storage.UserInfo{
		Username:     i.username,
		FriendlyName: i.friendlyName,
		Role:         i.perms,
		Permissions:  i.perms,
		Roles:        i.roles,
	}
}

func (i *roleBasedIdentity) Expiry() time.Time {
	return i.expiry
}

func (i *roleBasedIdentity) ExternalAuthProvider() authproviders.Provider {
	return i.authProvider
}

func (i *roleBasedIdentity) Attributes() map[string][]string {
	return i.attributes
}
