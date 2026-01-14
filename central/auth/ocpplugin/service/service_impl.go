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

var (
	_ central.TokenServiceServer = (*serviceImpl)(nil)

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		idcheck.SensorsOnly(): {
			central.TokenService_GetTokenForPermissionAndScope_FullMethodName,
		},
	})
)

type serviceImpl struct {
	issuer    tokens.Issuer
	roleStore roleDatastore.DataStore

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

func (s *serviceImpl) GetTokenForPermissionsAndScope(
	ctx context.Context,
	req *central.GetTokenForPermissionsAndScopeRequest,
) (*central.GetTokenForPermissionsAndScopeResponse, error) {
	targetRole, err := s.getRole(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "creating and storing target role")
	}
	roleName := targetRole.GetName()
	expiresAt, err := getExpiresAt(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "getting expiration time")
	}
	roxClaims := tokens.RoxClaims{
		RoleNames: []string{roleName},
		Name:      fmt.Sprintf("Generated claims for role %s expiring at %s", roleName, expiresAt.Format(time.RFC3339Nano)),
		ExpireAt:  &expiresAt,
	}
	tokenInfo, err := s.issuer.Issue(ctx, roxClaims)
	if err != nil {
		return nil, err
	}
	response := &central.GetTokenForPermissionsAndScopeResponse{
		TokenId:   "",
		Token:     tokenInfo.Token,
		Roles:     []string{roleName},
		IssuedAt:  protocompat.TimestampNow(),
		ExpiresAt: protocompat.ConvertTimeToTimestampOrNil(&expiresAt),
		Revoked:   false,
	}
	return response, nil
}

// region helpers

func (s *serviceImpl) getPermissionSet(ctx context.Context, req *central.GetTokenForPermissionsAndScopeRequest) (*storage.PermissionSet, error) {
	// TODO: Consider pruning the generated permission sets after some idle time.
	permissionSet := &storage.PermissionSet{
		ResourceToAccess: make(map[string]storage.Access),
		Traits:           &storage.Traits{Origin: storage.Traits_IMPERATIVE},
	}
	var b strings.Builder
	readResources := req.GetReadPermission()
	readAccessString := storage.Access_READ_ACCESS.String()
	for ix, resource := range readResources {
		if ix > 0 {
			b.WriteString(",")
		}
		b.WriteString(resource)
		b.WriteString(":")
		b.WriteString(readAccessString)
		permissionSet.ResourceToAccess[resource] = storage.Access_READ_ACCESS
	}
	permissionSetID := declarativeconfig.NewDeclarativePermissionSetUUID(b.String()).String()
	permissionSet.Id = permissionSetID
	permissionSet.Name = fmt.Sprintf("Generated permission set for %s", permissionSetID)
	err := s.roleStore.UpsertPermissionSet(ctx, permissionSet)
	if err != nil {
		return nil, errors.Wrap(err, "storing target role")
	}
	return permissionSet, nil
}

func (s *serviceImpl) getAccessScope(ctx context.Context, req *central.GetTokenForPermissionsAndScopeRequest) (*storage.SimpleAccessScope, error) {
	// TODO: Consider pruning the generated access scopes after some idle time.
	accessScope := &storage.SimpleAccessScope{
		Description: "",
		Rules:       &storage.SimpleAccessScope_Rules{},
		Traits:      &storage.Traits{Origin: storage.Traits_IMPERATIVE},
	}
	var b strings.Builder
	fullAccessClusters := make([]string, 0)
	clusterNamespaces := make([]*storage.SimpleAccessScope_Rules_Namespace, 0)
	for ix, clusterScope := range req.GetClusterScopes() {
		if ix > 0 {
			b.WriteString(";")
		}
		if clusterScope.FullClusterAccess {
			fullAccessClusters = append(fullAccessClusters, clusterScope.GetClusterName())
			b.WriteString(clusterScope.GetClusterName())
			b.WriteString(":*")
		} else {
			b.WriteString(clusterScope.GetClusterName())
			b.WriteString(":")
			for nsix, ns := range clusterScope.GetNamespace() {
				clusterNamespaces = append(clusterNamespaces, &storage.SimpleAccessScope_Rules_Namespace{
					ClusterName:   clusterScope.GetClusterName(),
					NamespaceName: ns,
				})
				if nsix > 0 {
					b.WriteString(",")
				}
				b.WriteString(ns)
			}
		}
	}
	accessScope.Rules.IncludedClusters = fullAccessClusters
	accessScope.Rules.IncludedNamespaces = clusterNamespaces
	accessScopeID := declarativeconfig.NewDeclarativeAccessScopeUUID(b.String()).String()
	accessScope.Id = accessScopeID
	accessScope.Name = fmt.Sprintf("Generated access scope for %s", accessScopeID)

	err := s.roleStore.UpsertAccessScope(ctx, accessScope)
	if err != nil {
		return nil, errors.Wrap(err, "storing access scope")
	}

	return accessScope, nil
}

func (s *serviceImpl) getRole(ctx context.Context, req *central.GetTokenForPermissionsAndScopeRequest) (*storage.Role, error) {
	// TODO: Consider pruning the generated roles after some idle time.
	ps, err := s.getPermissionSet(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "creating permission set for role")
	}
	as, err := s.getAccessScope(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "creating access scope for role")
	}
	psID := ps.GetId()
	asID := as.GetId()
	resultRole := &storage.Role{
		Name:            fmt.Sprintf("Generated role for PermissionSet %s and AccessScope %s", psID, asID),
		Description:     "Generated role for OCP console plugin",
		PermissionSetId: psID,
		AccessScopeId:   asID,
		Traits:          &storage.Traits{Origin: storage.Traits_IMPERATIVE},
	}
	// Store role in database
	return resultRole, nil
}

func getExpiresAt(_ context.Context, req *central.GetTokenForPermissionsAndScopeRequest) (time.Time, error) {
	duration, err := protocompat.DurationFromProto(req.GetValidFor())
	if err != nil {
		return time.Time{}, errors.Wrap(err, "converting requested token validity duration")
	}
	if duration <= 0 {
		return time.Time{}, errox.InvalidArgs.CausedBy("token validity duration should be positive")
	}
	return time.Now().Add(duration), nil
}

// endregion helpers
