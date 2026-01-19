package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	roleDatastore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/protocompat"
	"google.golang.org/grpc"
)

const (
	permissionSetNameFormat = "Generated permission set for %s"
	accessScopeNameFormat   = "Generated access scope for %s"
	roleNameFormat          = "Generated role for PermissionSet %s and AccessScope %s"

	primaryListSeparator   = ";"
	keyValueSeparator      = ":"
	secondaryListSeparator = ","
	clusterWildCard        = "*"
)

var (
	_ central.TokenServiceServer = (*serviceImpl)(nil)
	
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		idcheck.SensorsOnly(): {
			central.TokenService_GenerateTokenForPermissionsAndScope_FullMethodName,
		},
	})
)

type serviceImpl struct {
	issuer    tokens.Issuer
	roleStore roleDatastore.DataStore

	now func() time.Time

	central.UnimplementedTokenServiceServer
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	central.RegisterTokenServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return central.RegisterTokenServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GenerateTokenForPermissionsAndScope(
	ctx context.Context,
	req *central.GenerateTokenForPermissionsAndScopeRequest,
) (*central.GenerateTokenForPermissionsAndScopeResponse, error) {
	roleName, err := s.createRole(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "creating and storing target role")
	}
	expiresAt, err := s.getExpiresAt(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "getting expiration time")
	}
	claimName := fmt.Sprintf(
		"Generated claims for role %s expiring at %s",
		roleName,
		expiresAt.Format(time.RFC3339Nano),
	)
	roxClaims := tokens.RoxClaims{
		RoleNames: []string{roleName},
		Name:      claimName,
	}
	tokenInfo, err := s.issuer.Issue(ctx, roxClaims, tokens.WithExpiry(expiresAt))
	if err != nil {
		return nil, err
	}
	response := &central.GenerateTokenForPermissionsAndScopeResponse{
		Token: tokenInfo.Token,
	}
	return response, nil
}

// region helpers

var (
	generatedObjectTraits = &storage.Traits{Origin: storage.Traits_IMPERATIVE}
)

// createPermissionSet creates a dynamic permission set, granting the requested permissions.
// The returned information is the ID of the created permission set, or an error if any occurred
// in the creation process.
func (s *serviceImpl) createPermissionSet(
	ctx context.Context,
	req *central.GenerateTokenForPermissionsAndScopeRequest,
) (string, error) {
	// TODO: Consider pruning the generated permission sets after some idle time.
	permissionSet := &storage.PermissionSet{
		ResourceToAccess: make(map[string]storage.Access),
		Traits:           generatedObjectTraits.CloneVT(),
	}
	var b strings.Builder
	readResources := req.GetReadPermissions()
	readAccessString := storage.Access_READ_ACCESS.String()
	for ix, resource := range readResources {
		if ix > 0 {
			b.WriteString(primaryListSeparator)
		}
		b.WriteString(resource)
		b.WriteString(keyValueSeparator)
		b.WriteString(readAccessString)
		permissionSet.ResourceToAccess[resource] = storage.Access_READ_ACCESS
	}
	permissionSetID := declarativeconfig.NewDeclarativePermissionSetUUID(b.String()).String()
	permissionSet.Id = permissionSetID
	permissionSet.Name = fmt.Sprintf(permissionSetNameFormat, permissionSetID)
	err := s.roleStore.UpsertPermissionSet(ctx, permissionSet)
	if err != nil {
		return "", errors.Wrap(err, "storing permission set")
	}
	return permissionSet.GetId(), nil
}

// createAccessScope creates a dynamic access scope, granting the requested scope.
// The returned information is the identifier of the created access scope,
// or an error if any occurred in the creation process.
func (s *serviceImpl) createAccessScope(
	ctx context.Context,
	req *central.GenerateTokenForPermissionsAndScopeRequest,
) (string, error) {
	// TODO: Consider pruning the generated access scopes after some idle time.
	accessScope := &storage.SimpleAccessScope{
		Description: "",
		Rules:       &storage.SimpleAccessScope_Rules{},
		Traits:      generatedObjectTraits.CloneVT(),
	}
	var b strings.Builder
	fullAccessClusters := make([]string, 0)
	clusterNamespaces := make([]*storage.SimpleAccessScope_Rules_Namespace, 0)
	for ix, clusterScope := range req.GetClusterScopes() {
		if ix > 0 {
			b.WriteString(primaryListSeparator)
		}
		b.WriteString(clusterScope.GetClusterName())
		b.WriteString(keyValueSeparator)
		if clusterScope.GetFullClusterAccess() {
			fullAccessClusters = append(fullAccessClusters, clusterScope.GetClusterName())
			b.WriteString(clusterWildCard)
		} else {
			for namespaceIndex, namespace := range clusterScope.GetNamespaces() {
				clusterNamespaces = append(clusterNamespaces, &storage.SimpleAccessScope_Rules_Namespace{
					ClusterName:   clusterScope.GetClusterName(),
					NamespaceName: namespace,
				})
				if namespaceIndex > 0 {
					b.WriteString(secondaryListSeparator)
				}
				b.WriteString(namespace)
			}
		}
	}
	accessScope.Rules.IncludedClusters = fullAccessClusters
	accessScope.Rules.IncludedNamespaces = clusterNamespaces
	accessScopeID := declarativeconfig.NewDeclarativeAccessScopeUUID(b.String()).String()
	accessScope.Id = accessScopeID
	accessScope.Name = fmt.Sprintf(accessScopeNameFormat, accessScopeID)

	err := s.roleStore.UpsertAccessScope(ctx, accessScope)
	if err != nil {
		return "", errors.Wrap(err, "storing access scope")
	}

	return accessScope.GetId(), nil
}

// createRole creates a dynamic role, granting the requested permissions and scope.
// The returned information is the name of the created role, or an error if any occurred
// in the creation process.
func (s *serviceImpl) createRole(
	ctx context.Context,
	req *central.GenerateTokenForPermissionsAndScopeRequest,
) (string, error) {
	// TODO: Consider pruning the generated roles after some idle time.
	permissionSetID, err := s.createPermissionSet(ctx, req)
	if err != nil {
		return "", errors.Wrap(err, "creating permission set for role")
	}
	accessScopeID, err := s.createAccessScope(ctx, req)
	if err != nil {
		return "", errors.Wrap(err, "creating access scope for role")
	}
	resultRole := &storage.Role{
		Name:            fmt.Sprintf(roleNameFormat, permissionSetID, accessScopeID),
		Description:     "Generated role for OCP console plugin",
		PermissionSetId: permissionSetID,
		AccessScopeId:   accessScopeID,
		Traits:          generatedObjectTraits.CloneVT(),
	}
	err = s.roleStore.UpsertRole(ctx, resultRole)
	if err != nil {
		return "", errors.Wrap(err, "storing role")
	}

	return resultRole.GetName(), nil
}

func (s *serviceImpl) getExpiresAt(
	_ context.Context,
	req *central.GenerateTokenForPermissionsAndScopeRequest,
) (time.Time, error) {
	duration, err := protocompat.DurationFromProto(req.GetValidFor())
	if err != nil {
		return time.Time{}, errors.Wrap(err, "converting requested token validity duration")
	}
	if duration <= 0 {
		return time.Time{}, errox.InvalidArgs.CausedBy("token validity duration should be positive")
	}
	return s.now().Add(duration), nil
}

// endregion helpers
