package basic

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/sac/resources"
)

func contextWithBasicIdentityForTest(
	t *testing.T,
	name string,
	roles []permissions.ResolvedRole,
	authProvider authproviders.Provider,
) context.Context {
	contextIdentity := &identity{
		username:      name,
		resolvedRoles: roles,
		authProvider:  authProvider,
	}
	return authn.ContextWithIdentity(context.Background(), contextIdentity, t)
}

// ContextWithAdminIdentity returns a context enriched with an Identity
// that is granted full admin role.
func ContextWithAdminIdentity(
	t *testing.T,
	authProvider authproviders.Provider,
) context.Context {
	adminPermissions := make(map[string]storage.Access)
	for _, p := range resources.ListAll() {
		permissionName := string(p)
		adminPermissions[permissionName] = storage.Access_READ_WRITE_ACCESS
	}
	adminRole := &testRole{
		name:        accesscontrol.Admin,
		permissions: adminPermissions,
		accessScope: accessScopeIncludeAll,
	}
	roles := []permissions.ResolvedRole{adminRole}
	return contextWithBasicIdentityForTest(t, accesscontrol.Admin, roles, authProvider)
}

// ContextWithNoAccessIdentity returns a context enriched with an Identity
// that is granted a role with neither permissions nor scope.
func ContextWithNoAccessIdentity(
	t *testing.T,
	authProvider authproviders.Provider,
) context.Context {
	const noAccessName = "No Access"
	noAccessPermissions := make(map[string]storage.Access)
	for _, p := range resources.ListAll() {
		permissionName := string(p)
		noAccessPermissions[permissionName] = storage.Access_NO_ACCESS
	}
	adminRole := &testRole{
		name:        noAccessName,
		permissions: noAccessPermissions,
		accessScope: accessScopeIncludeAll,
	}
	roles := []permissions.ResolvedRole{adminRole}
	return contextWithBasicIdentityForTest(t, noAccessName, roles, authProvider)
}

// ContextWithNoneIdentity returns a context enriched with an Identity
// that is granted only the None role.
func ContextWithNoneIdentity(
	t *testing.T,
	authProvider authproviders.Provider,
) context.Context {
	noneRole := &testRole{
		name:        accesscontrol.None,
		accessScope: accessScopeIncludeAll,
	}
	roles := []permissions.ResolvedRole{noneRole}
	return contextWithBasicIdentityForTest(t, accesscontrol.None, roles, authProvider)
}

// ContextWithNoRoleIdentity returns a context enriched with an Identity
// that is granted full admin role.
func ContextWithNoRoleIdentity(
	t *testing.T,
	authProvider authproviders.Provider,
) context.Context {
	const noRoleName = "No Role"
	roles := []permissions.ResolvedRole{}
	return contextWithBasicIdentityForTest(t, noRoleName, roles, authProvider)
}

type testRole struct {
	name        string
	permissions map[string]storage.Access
	accessScope *storage.SimpleAccessScope
}

// GetRoleName returns the name of the role.
func (r *testRole) GetRoleName() string {
	return r.name
}

// GetPermissions returns the granted permissions for the role.
func (r *testRole) GetPermissions() map[string]storage.Access {
	return r.permissions
}

// GetAccessScope returns the access scope associated with the role.
func (r *testRole) GetAccessScope() *storage.SimpleAccessScope {
	return r.accessScope
}

var (
	// accessScopeIncludeAll gives access to all resources. It is checked by ID, as
	// Rules cannot represent unrestricted scope.
	accessScopeIncludeAll = &storage.SimpleAccessScope{
		Id:          accesscontrol.DefaultAccessScopeIDs[accesscontrol.UnrestrictedAccessScope],
		Name:        accesscontrol.UnrestrictedAccessScope,
		Description: "Access to all clusters and namespaces",
		Traits: &storage.Traits{
			Origin: storage.Traits_DEFAULT,
		},
	}
)
