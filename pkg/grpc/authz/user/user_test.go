package user

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_permissionChecker_Authorized(t *testing.T) {
	clusterScopedResource := permissions.ResourceMetadata{
		Resource: "dummy-1", Scope: permissions.ClusterScope,
	}
	nsScopedResource := permissions.ResourceMetadata{
		Resource: "dummy-2", Scope: permissions.NamespaceScope,
	}
	globalScopedResource := permissions.ResourceMetadata{
		Resource: "dummy-3", Scope: permissions.GlobalScope,
	}

	testRole := roletest.NewResolvedRoleWithDenyAll("Dummy", nil)

	id := mocks.NewMockIdentity(gomock.NewController(t))
	ctx := authn.ContextWithIdentity(context.Background(), id, t)
	id.EXPECT().Roles().Return([]permissions.ResolvedRole{testRole}).AnyTimes()
	id.EXPECT().Permissions().Return(map[string]storage.Access{
		string(clusterScopedResource.Resource): storage.Access_READ_WRITE_ACCESS,
	}).AnyTimes()

	// There is a slight difference between idWithNoPermissions and
	// idWithEmptyPermissions in the way what Permissions() returns.
	// There is an assumption that identity implementations use nil when no
	// Roles are associated with the identity and empty map when Role(s) has
	// no permissions.

	idWithNoPermissions := mocks.NewMockIdentity(gomock.NewController(t))
	ctxWithNoPermissions := authn.ContextWithIdentity(context.Background(), idWithNoPermissions, t)
	idWithNoPermissions.EXPECT().Roles().Return([]permissions.ResolvedRole{testRole}).AnyTimes()
	idWithNoPermissions.EXPECT().Permissions().Return(nil).AnyTimes()

	idWithEmptyPermissions := mocks.NewMockIdentity(gomock.NewController(t))
	ctxWithEmptyPermissions := authn.ContextWithIdentity(context.Background(), idWithEmptyPermissions, t)
	idWithEmptyPermissions.EXPECT().Roles().Return([]permissions.ResolvedRole{testRole}).AnyTimes()
	idWithEmptyPermissions.EXPECT().Permissions().Return(map[string]storage.Access{}).AnyTimes()

	tests := []struct {
		name                string
		requiredPermissions []permissions.ResourceWithAccess
		ctx                 context.Context
		err                 error
	}{
		{
			name: "no ID in context => error",
			ctx:  context.Background(),
			err:  errox.NoCredentials,
		},
		{
			name: "permissions equal access => no error",
			requiredPermissions: []permissions.ResourceWithAccess{{
				Resource: clusterScopedResource, Access: storage.Access_READ_WRITE_ACCESS,
			}},
			ctx: ctx,
		},
		{
			name:                "authenticated with no permissions => no error",
			requiredPermissions: []permissions.ResourceWithAccess{},
			ctx:                 ctxWithEmptyPermissions,
		},
		{
			name:                "authenticated with no permissions and deny all scope => no error",
			requiredPermissions: []permissions.ResourceWithAccess{},
			ctx:                 sac.WithNoAccess(ctxWithEmptyPermissions),
		},
		{
			name: "built-in scoped authz check permissions not sufficient permissions",
			requiredPermissions: []permissions.ResourceWithAccess{{
				Resource: clusterScopedResource, Access: storage.Access_READ_WRITE_ACCESS,
			}, {
				Resource: nsScopedResource, Access: storage.Access_READ_ACCESS,
			}},
			ctx: sac.WithNoAccess(ctx),
			err: errox.NotAuthorized,
		},
		{
			name: "built-in scoped authz check permissions",
			requiredPermissions: []permissions.ResourceWithAccess{{
				Resource: clusterScopedResource, Access: storage.Access_READ_WRITE_ACCESS,
			}},
			ctx: sac.WithNoAccess(ctx),
		},
		{
			name: "built-in scoped authz check permissions but nil permissions in ID",
			requiredPermissions: []permissions.ResourceWithAccess{{
				Resource: clusterScopedResource, Access: storage.Access_READ_WRITE_ACCESS,
			}},
			ctx: sac.WithNoAccess(ctxWithNoPermissions),
			err: errox.NoCredentials,
		},
		{
			name: "built-in global scoped authz check permissions but nil permissions in ID",
			requiredPermissions: []permissions.ResourceWithAccess{{
				Resource: globalScopedResource, Access: storage.Access_READ_WRITE_ACCESS,
			}},
			ctx: sac.WithNoAccess(ctxWithNoPermissions),
			err: errox.NoCredentials,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := With(tt.requiredPermissions...)
			err := p.Authorized(tt.ctx, "not used")
			assert.ErrorIs(t, err, tt.err)

			// Once authentication is successful, Authenticated authorizer shall
			// not return errox.NotAuthorized in contrast to With authorizer;
			// otherwise the two should behave the same.
			a := Authenticated()
			err2 := a.Authorized(tt.ctx, "not used")
			if errors.Is(err, errox.NotAuthorized) {
				assert.ErrorIs(t, err2, nil)
			} else {
				assert.ErrorIs(t, err2, tt.err)
			}
		})
	}
}

func TestEvaluateAgainstPermissions(t *testing.T) {
	type expectation struct {
		view   bool
		modify bool
	}
	writeAccessibleResource := permissions.ResourceMetadata{
		Resource: permissions.Resource("writeaccessible"),
	}
	readAccessibleResource := permissions.ResourceMetadata{
		Resource: permissions.Resource("readaccessible"),
	}
	forbiddenResource := permissions.ResourceMetadata{
		Resource: permissions.Resource("forbidden"),
	}

	perms := utils.FromResourcesWithAccess(
		permissions.Modify(writeAccessibleResource),
		permissions.View(readAccessibleResource),
	)

	expectations := map[permissions.ResourceMetadata]expectation{
		writeAccessibleResource: {view: true, modify: true},
		readAccessibleResource:  {view: true},
		forbiddenResource:       {},
	}

	for resourceMetadata, exp := range expectations {
		t.Run(fmt.Sprintf("resource: %s", resourceMetadata), func(t *testing.T) {
			assert.Equal(t, exp.view, evaluateAgainstPermissions(perms, permissions.View(resourceMetadata)))
			assert.Equal(t, exp.modify, evaluateAgainstPermissions(perms, permissions.Modify(resourceMetadata)))
		})
	}
}
