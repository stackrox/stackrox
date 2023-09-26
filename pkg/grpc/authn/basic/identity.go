package basic

import (
	"strings"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/timeutil"
)

var _ authn.Identity = (*identity)(nil)
var log = logging.LoggerForModule()

// IsBasicIdentity returns whether or not the input Identity is a basic identity.
func IsBasicIdentity(id authn.Identity) bool {
	_, isBasic := id.(Identity)
	return isBasic
}

// Basic identity implementation.
type identity struct {
	username      string
	resolvedRoles []permissions.ResolvedRole
	authProvider  authproviders.Provider
}

func (i identity) UID() string {
	return i.username
}

func (i identity) FriendlyName() string {
	return i.username
}

func (i identity) FullName() string {
	return i.username
}

func (i identity) Permissions() map[string]storage.Access {
	return utils.NewUnionPermissions(i.resolvedRoles)
}

func (i identity) Roles() []permissions.ResolvedRole {
	return i.resolvedRoles
}

func (i identity) Service() *storage.ServiceIdentity {
	return nil
}

func (i identity) TenantID() string {
	// Assume username is in correct format
	tenant, _, _ := strings.Cut(i.username, "-")
	return tenant
}

func (i identity) User() *storage.UserInfo {
	return &storage.UserInfo{
		Username:    i.username,
		Permissions: &storage.UserInfo_ResourceToAccess{ResourceToAccess: i.Permissions()},
		Roles:       utils.ExtractRolesForUserInfo(i.resolvedRoles),
	}
}

func (i identity) ValidityPeriod() (time.Time, time.Time) {
	return time.Time{}, timeutil.MaxProtoValid
}

func (i identity) ExternalAuthProvider() authproviders.Provider {
	return i.authProvider
}

func (i identity) isBasicAuthIdentity() {}

func (i identity) AsExternalUser() *tokens.ExternalUserClaim {
	return &tokens.ExternalUserClaim{
		UserID:     i.username,
		Attributes: i.Attributes(),
	}
}

func (i identity) Attributes() map[string][]string {
	return map[string][]string{
		"username":  {i.username},
		"role":      utils.RoleNames(i.resolvedRoles),
		"tenant_id": {i.TenantID()},
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
