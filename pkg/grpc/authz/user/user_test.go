package user

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/authz/internal/permissioncheck"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stretchr/testify/assert"
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

	idWithNoPermissions := mocks.NewMockIdentity(gomock.NewController(t))
	ctxWithNoPermissions := authn.ContextWithIdentity(context.Background(), idWithNoPermissions, t)
	idWithNoPermissions.EXPECT().Roles().Return([]permissions.ResolvedRole{testRole}).AnyTimes()
	idWithNoPermissions.EXPECT().Permissions().Return(nil).AnyTimes()

	contextWithPermissionCheck, _ := permissioncheck.ContextWithPermissionCheck()

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
			name: "ErrPermissionCheckOnly",
			ctx:  contextWithPermissionCheck,
			err:  permissioncheck.ErrPermissionCheckOnly,
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
