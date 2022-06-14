package authz

import (
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/stackrox/stackrox/pkg/auth/permissions"
	"github.com/stackrox/stackrox/pkg/grpc/authz/internal/permissioncheck"
)

// GetPermissionMapForServiceMethod retrieves a PermissionMap of all permissions checked
// by a service method.
func GetPermissionMapForServiceMethod(srv interface{}, fullMethodName string) []permissions.ResourceWithAccess {
	if authFunc, ok := srv.(grpc_auth.ServiceAuthFuncOverride); ok {
		ctx, perms := permissioncheck.ContextWithPermissionCheck()
		_, _ = authFunc.AuthFuncOverride(ctx, fullMethodName)
		return *perms
	}
	return nil
}
