package transitional

import (
	"context"

	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

// VerifySACScopeChecksInterceptor is a GRPC unary interceptor that verifies that the permissions
// checked for by scoped access control are at least as strong as the permissions governing access
// to the service method.
func VerifySACScopeChecksInterceptor(ctx context.Context, req interface{}, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	scc := sac.GlobalAccessScopeChecker(ctx).Core()

	recordingSCC := newPermissionRecordingSCC(scc)

	newCtx := sac.WithGlobalAccessScopeChecker(ctx, recordingSCC)

	resp, err := handler(newCtx, req)
	if err != nil {
		return nil, err
	}

	serviceMethodPerms := authz.GetPermissionMapForServiceMethod(serverInfo.Server, serverInfo.FullMethod)
	usedPerms := recordingSCC.UsedPermissions()
	if !serviceMethodPerms.IsLessOrEqual(usedPerms) {
		log.Errorf("Method %s required permissions %v, but scoped access control only checked for permissions %v", serverInfo.FullMethod, serviceMethodPerms, usedPerms)
	}

	return resp, nil
}
