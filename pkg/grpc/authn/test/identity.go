package test

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stackrox/rox/pkg/timeutil"
)

var _ authn.Identity = (*identity)(nil)

// Test identity implementation.
type identity struct {
	username      string
	resolvedRoles []permissions.ResolvedRole
}

// NewTestIdentity creates a test identity.
func NewTestIdentity(userName string, _ *testing.T) *identity {
	return &identity{username: userName}
}

func (i *identity) Context() context.Context {
	return authn.ContextWithIdentity(context.Background(), i, nil)
}

func (i *identity) AddRole(resource permissions.Resource, access storage.Access, as *storage.SimpleAccessScope) *identity {
	i.resolvedRoles = append(i.resolvedRoles, roletest.NewResolvedRole("test",
		map[string]storage.Access{string(resource): access}, as))
	return i
}

func (i *identity) UID() string {
	return i.username
}

func (i *identity) FriendlyName() string {
	return i.username
}

func (i *identity) FullName() string {
	return i.username
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

func (i *identity) User() *storage.UserInfo {
	return &storage.UserInfo{
		Username:    i.username,
		Permissions: &storage.UserInfo_ResourceToAccess{ResourceToAccess: i.Permissions()},
		Roles:       utils.ExtractRolesForUserInfo(i.resolvedRoles),
	}
}

func (i *identity) ValidityPeriod() (time.Time, time.Time) {
	return time.Time{}, timeutil.MaxProtoValid
}

func (i *identity) ExternalAuthProvider() authproviders.Provider {
	return nil
}

func (i *identity) Attributes() map[string][]string {
	return map[string][]string{
		"username": {i.username},
		"role":     utils.RoleNames(i.resolvedRoles),
	}
}
