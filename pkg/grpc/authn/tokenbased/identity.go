package tokenbased

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/grpc/authn"
)

var _ authn.Identity = (*roleBasedIdentity)(nil)

type roleBasedIdentity struct {
	uid           string
	username      string
	friendlyName  string
	fullName      string
	resolvedRoles []permissions.ResolvedRole
	expiry        time.Time
	attributes    map[string][]string
	authProvider  authproviders.Provider
}

func (i *roleBasedIdentity) TenantID() string {
	if tenant, exists := i.attributes["tenant_id"]; exists {
		return tenant[0]
	}
	panic("No tenant available")
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

func (i *roleBasedIdentity) Permissions() map[string]storage.Access {
	return utils.NewUnionPermissions(i.resolvedRoles)
}

func (i *roleBasedIdentity) Roles() []permissions.ResolvedRole {
	return i.resolvedRoles
}

func (i *roleBasedIdentity) Service() *storage.ServiceIdentity {
	return nil
}

func (i *roleBasedIdentity) User() *storage.UserInfo {
	return &storage.UserInfo{
		Username:     i.username,
		FriendlyName: i.friendlyName,
		Permissions:  &storage.UserInfo_ResourceToAccess{ResourceToAccess: i.Permissions()},
		Roles:        utils.ExtractRolesForUserInfo(i.resolvedRoles),
	}
}

func (i *roleBasedIdentity) ValidityPeriod() (time.Time, time.Time) {
	return time.Time{}, i.expiry
}

func (i *roleBasedIdentity) ExternalAuthProvider() authproviders.Provider {
	return i.authProvider
}

func (i *roleBasedIdentity) Attributes() map[string][]string {
	return i.attributes
}
