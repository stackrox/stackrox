package transitional

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()

	logMessages      = set.NewStringSet()
	logMessagesMutex sync.Mutex

	searchResources = []permissions.ResourceHandle{
		resources.Alert,
		resources.Compliance,
		resources.Deployment,
		resources.Image,
		resources.K8sRole,
		resources.K8sRoleBinding,
		resources.Namespace,
		resources.Node,
		resources.Policy,
		resources.Secret,
		resources.ServiceAccount,
	}

	staticWhitelist = map[string][]permissions.ResourceHandle{
		// Access control enforcement for Search in the SAC world is dynamic - which SAC scopes are checked for depends
		// on what you search for.
		"/v1.SearchService/Search":       searchResources,
		"/v1.SearchService/Autocomplete": searchResources,
		// No actual leakage of data, only static data returned
		"/v1.SearchService/Options": searchResources,
		// No actual leakage of data, only static data returned
		"/v1.ComplianceService/GetStandards": {resources.Compliance},
		"/v1.ComplianceService/GetStandard":  {resources.Compliance},
		// No actual data leakage, and virtually impossible to translate to scoped access.
		"/v1.NetworkPolicyService/GetNetworkGraphEpoch": {resources.NetworkPolicy},
		// This does not actually return process indicators.
		"/v1.DeploymentService/ListDeploymentsWithProcessInfo": {resources.Indicator},
		// K8sSubject is a mostly meaningless resource. Subjects in the sense of StackRox (users and groups) are not
		// stored by k8s but only referenced in role bindings. Since these APIs are implemented by searching for all
		// role bindings and then extracting the subjects, SAC checks for roles and role bindings that happen are
		// sufficient (and actually stronger).
		"/v1.RbacService/GetSubject":   {resources.K8sSubject},
		"/v1.RbacService/ListSubjects": {resources.K8sSubject},
	}
)

// VerifySACScopeChecksInterceptor is a GRPC unary interceptor that verifies that the permissions
// checked for by scoped access control are at least as strong as the permissions governing access
// to the service method.
func VerifySACScopeChecksInterceptor(ctx context.Context, req interface{}, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// We cannot validate the built-in scoped authorizer's permission checks
	// because it does not check permissions via ScopeCheckerCore::TryAllowed().
	if !sac.IsContextPluginScopedAuthzEnabled(ctx) {
		return handler(ctx, req)
	}

	scc := sac.GlobalAccessScopeChecker(ctx).Core()

	recordingSCC := newPermissionRecordingSCC(scc)

	newCtx := sac.WithGlobalAccessScopeChecker(ctx, recordingSCC)

	resp, err := handler(newCtx, req)
	if err != nil {
		return nil, err
	}

	serviceMethodPerms := authz.GetPermissionMapForServiceMethod(serverInfo.Server, serverInfo.FullMethod)
	serviceMethodPermSet := make(permissions.PermissionMap)
	for _, resourceAccess := range serviceMethodPerms {
		// Skip resources that do not bypass SAC legacy auth - these will be checked by the authz middleware.
		if resourceAccess.Resource.PerformLegacyAuthForSAC() {
			continue
		}
		serviceMethodPermSet.Add(resourceAccess.Resource, resourceAccess.Access)
	}

	// Apply process baseline rules.
	for _, wlResource := range staticWhitelist[serverInfo.FullMethod] {
		delete(serviceMethodPermSet, wlResource.GetResource())
	}

	usedPerms := recordingSCC.UsedPermissions()
	if serviceMethodPermSet.IsLessOrEqual(usedPerms) {
		return resp, nil
	}

	// If a response contains no data, we can skip all checks for accessing scoped resources with read-only level.
	// Note: this does not affect APIs that return an empty message anyway.
	if pb, ok := resp.(proto.Message); ok {
		if _, isEmpty := pb.(*v1.Empty); !isEmpty {
			bytes, err := proto.Marshal(pb)
			if err == nil && len(bytes) == 0 {
				for _, resourceAndAccess := range serviceMethodPerms {
					if resourceAndAccess.Access == storage.Access_READ_ACCESS {
						delete(serviceMethodPermSet, resourceAndAccess.Resource.Resource)
					}
				}
			}
		}
	}

	if !serviceMethodPermSet.IsLessOrEqual(usedPerms) {
		logMsg := fmt.Sprintf("Method %s required permissions %v, but scoped access control only checked for permissions %v", serverInfo.FullMethod, serviceMethodPerms, usedPerms)

		added := false
		concurrency.WithLock(&logMessagesMutex, func() {
			added = logMessages.Add(logMsg)
		})
		if added {
			log.Error(logMsg)
		}
	}

	return resp, nil
}
