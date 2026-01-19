package service

import (
	"context"
	"fmt"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/auth/tokens"
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
	issuer tokens.Issuer

	roleManager *roleManager

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
	roleName, err := s.roleManager.createRole(ctx, req)
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
