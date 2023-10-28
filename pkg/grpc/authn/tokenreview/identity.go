package tokenreview

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/timeutil"
)

var _ authn.Identity = (*k8sBasedIdentity)(nil)

type k8sBasedIdentity struct {
	uid           string
	username      string
	resolvedRoles []permissions.ResolvedRole
	attributes    map[string][]string
}

func (i *k8sBasedIdentity) UID() string {
	return i.uid
}

func (i *k8sBasedIdentity) FriendlyName() string {
	return i.username
}

func (i *k8sBasedIdentity) FullName() string {
	return i.username
}

func (i *k8sBasedIdentity) Permissions() map[string]storage.Access {
	return utils.NewUnionPermissions(i.resolvedRoles)
}

func (i *k8sBasedIdentity) Roles() []permissions.ResolvedRole {
	return i.resolvedRoles
}

func (i *k8sBasedIdentity) Service() *storage.ServiceIdentity {
	return nil
}

func (i *k8sBasedIdentity) User() *storage.UserInfo {
	return &storage.UserInfo{
		Username:     i.username,
		FriendlyName: i.username,
		Permissions:  &storage.UserInfo_ResourceToAccess{ResourceToAccess: i.Permissions()},
		Roles:        utils.ExtractRolesForUserInfo(i.resolvedRoles),
	}
}

func (i *k8sBasedIdentity) ValidityPeriod() (time.Time, time.Time) {
	return time.Time{}, timeutil.MaxProtoValid
}

func (i *k8sBasedIdentity) ExternalAuthProvider() authproviders.Provider {
	return nil
}

func (i *k8sBasedIdentity) Attributes() map[string][]string {
	return i.attributes
}
