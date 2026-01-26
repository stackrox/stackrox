package service

import (
	"context"
	"fmt"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

const (
	claimNameFormat = "Generated claims for role %s expiring at %s"

	// rbacObjectsGraceExpiration expands expired RBAC objects lifetime to allow
	// requests complete even if the token expires during requests handling.
	rbacObjectsGraceExpiration = 2 * time.Minute
)

var (
	_ v1.TokenServiceServer = (*serviceImpl)(nil)

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		idcheck.SensorsOnly(): {
			v1.TokenService_GenerateTokenForPermissionsAndScope_FullMethodName,
		},
	})

	clusterReadContext = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster),
		),
	)

	errBadExpirationValue = errox.InvalidArgs.New("bad expiration timestamp")
)

type serviceImpl struct {
	issuer tokens.Issuer

	roleManager *roleManager

	now func() time.Time

	v1.UnimplementedTokenServiceServer
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterTokenServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterTokenServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GenerateTokenForPermissionsAndScope(
	ctx context.Context,
	req *v1.GenerateTokenForPermissionsAndScopeRequest,
) (*v1.GenerateTokenForPermissionsAndScopeResponse, error) {
	// Calculate expiry first so we can set it on the RBAC objects.
	expiresAt, err := s.getExpiresAt(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "getting expiration time")
	}
	traits, err := generateTraitsWithExpiry(expiresAt.Add(rbacObjectsGraceExpiration))
	if err != nil {
		return nil, errBadExpirationValue.CausedBy(err)
	}
	// Create the role with the same expiry as the token, so pruning can clean
	// it up.
	roleName, err := s.roleManager.createRole(ctx, req, traits)
	if err != nil {
		return nil, errors.Wrap(err, "creating and storing target role")
	}
	claimName := fmt.Sprintf(claimNameFormat, roleName, expiresAt.Format(time.RFC3339Nano))
	roxClaims := tokens.RoxClaims{
		RoleNames: []string{roleName},
		Name:      claimName,
	}
	tokenInfo, err := s.issuer.Issue(ctx, roxClaims, tokens.WithExpiry(expiresAt))
	if err != nil {
		return nil, err
	}
	response := &v1.GenerateTokenForPermissionsAndScopeResponse{
		Token: tokenInfo.Token,
	}
	go trackRequest(authn.IdentityFromContextOrNil(ctx), req)
	return response, nil
}

func (s *serviceImpl) getExpiresAt(
	_ context.Context,
	req *v1.GenerateTokenForPermissionsAndScopeRequest,
) (time.Time, error) {
	duration, err := protocompat.DurationFromProto(req.GetLifetime())
	if err != nil {
		return time.Time{}, errox.InvalidArgs.CausedByf("converting requested token validity duration: %v", err)
	}
	if duration <= 0 {
		return time.Time{}, errox.InvalidArgs.CausedBy("token validity duration should be positive")
	}
	return s.now().Add(duration), nil
}
