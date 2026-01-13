package service

import (
	"context"
	"fmt"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
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
	issuer tokens.Issuer

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
	targetRole := getRole(ctx, req)
	roleName := targetRole.GetName()
	expiresAt := getExpiresAt(ctx, req)
	roxClaims := tokens.RoxClaims{
		RoleNames: []string{roleName},
		Name:      "",
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

func getPermissionSet(_ context.Context, _ *central.GetTokenForPermissionsAndScopeRequest) *storage.PermissionSet {
	// Generate permission set and store it in database
	// The object ID should be a deterministic UUID based on the requested permissions
	return &storage.PermissionSet{
		Id: accesscontrol.DefaultPermissionSetIDs[accesscontrol.Analyst],
	}
}

func getAccessScope(_ context.Context, _ *central.GetTokenForPermissionsAndScopeRequest) *storage.SimpleAccessScope {
	// Generate simple access scope and store it in database.
	// The object ID should be a deterministic UUID based on the effective access scope tree obtained from the input.
	return role.AccessScopeIncludeAll
}

func getRole(ctx context.Context, req *central.GetTokenForPermissionsAndScopeRequest) *storage.Role {
	ps := getPermissionSet(ctx, req)
	as := getAccessScope(ctx, req)
	psID := ps.GetId()
	asID := as.GetId()
	role := &storage.Role{
		Name:            fmt.Sprintf("Generated role for PermissionSet %s and AccessScope %s", psID, asID),
		Description:     "Generated role for OCP console plugin",
		PermissionSetId: psID,
		AccessScopeId:   asID,
		Traits:          &storage.Traits{Origin: storage.Traits_IMPERATIVE},
	}
	// Store role in database
	return role
}

func getExpiresAt(_ context.Context, req *central.GetTokenForPermissionsAndScopeRequest) time.Time {
	duration, err := protocompat.DurationFromProto(req.GetValidFor())
	if err != nil {
		duration = 1 * time.Minute
	}
	return time.Now().Add(duration)
}

// endregion helpers
