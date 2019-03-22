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
	role         *storage.Role
	expiry       time.Time
}

func (i *roleBasedIdentity) UID() string {
	return i.uid
}

func (i *roleBasedIdentity) FriendlyName() string {
	return i.friendlyName
}

func (i *roleBasedIdentity) Role() *storage.Role {
	return i.role
}

func (i *roleBasedIdentity) Service() *storage.ServiceIdentity {
	return nil
}

func (i *roleBasedIdentity) User() *storage.UserInfo {
	return &storage.UserInfo{
		Username:     i.username,
		FriendlyName: i.friendlyName,
		Role:         i.role,
	}
}

func (i *roleBasedIdentity) Expiry() time.Time {
	return i.expiry
}

func (i *roleBasedIdentity) ExternalAuthProvider() authproviders.Provider {
	return nil
}
