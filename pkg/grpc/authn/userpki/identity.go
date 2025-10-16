package userpki

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/mtls"
)

var _ authn.Identity = (*identity)(nil)

type identity struct {
	info          mtls.CertInfo
	provider      authproviders.Provider
	resolvedRoles []permissions.ResolvedRole
	attributes    map[string][]string
}

func (i *identity) Attributes() map[string][]string {
	return i.attributes
}

func (i *identity) FriendlyName() string {
	return i.info.Subject.CommonName
}

func (i *identity) FullName() string {
	return i.info.Subject.CommonName
}

func (i *identity) User() *storage.UserInfo {
	ur := &storage.UserInfo_ResourceToAccess{}
	ur.SetResourceToAccess(i.Permissions())
	userInfo := &storage.UserInfo{}
	userInfo.SetFriendlyName(i.info.Subject.CommonName)
	userInfo.SetPermissions(ur)
	userInfo.SetRoles(utils.ExtractRolesForUserInfo(i.resolvedRoles))
	return userInfo
}

func (i *identity) Permissions() map[string]storage.Access {
	return utils.NewUnionPermissions(i.resolvedRoles)
}

func (i *identity) Roles() []permissions.ResolvedRole {
	return i.resolvedRoles
}

func (i *identity) Service() *storage.ServiceIdentity {
	return nil
}

func (i identity) ValidityPeriod() (time.Time, time.Time) {
	return time.Time{}, i.info.NotAfter
}

func (i *identity) ExternalAuthProvider() authproviders.Provider {
	return i.provider
}

func (i *identity) UID() string {
	return userID(i.info)
}

func userID(info mtls.CertInfo) string {
	return "userpki:" + info.CertFingerprint
}
