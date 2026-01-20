package service

import (
	"context"
	"fmt"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/telemetry/centralclient"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
	"google.golang.org/grpc"
)

const (
	claimNameFormat = "Generated claims for role %s expiring at %s"
)

var (
	_ v1.TokenServiceServer = (*serviceImpl)(nil)

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		idcheck.SensorsOnly(): {
			v1.TokenService_GenerateTokenForPermissionsAndScope_FullMethodName,
		},
	})
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
	roleName, err := s.roleManager.createRole(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "creating and storing target role")
	}
	expiresAt, err := s.getExpiresAt(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "getting expiration time")
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

func trackRequest(id authn.Identity, req *v1.GenerateTokenForPermissionsAndScopeRequest) {
	var client telemeter.Option
	// Track from the name of the secured cluster, if the caller is sensor.
	switch id.Service().GetType() {
	case storage.ServiceType_SENSOR_SERVICE:
		clientID := id.Service().GetId()
		cluster, _, _ := clusterDS.Singleton().GetCluster(context.Background(), clientID)
		client = telemeter.WithClient(clientID, "Secured Cluster", cluster.GetMainImage())
	case storage.ServiceType_UNKNOWN_SERVICE:
		client = telemeter.WithClient(id.UID(), "OCP Token Client", "")
	default:
		client = telemeter.WithClient(id.Service().GetId(), id.Service().GetType().String(), "")
	}

	req.GetClusterScopes()[0].GetNamespaces()
	maxNamespaces := 0
	fullClusterAccess := 0
	for _, cs := range req.GetClusterScopes() {
		maxNamespaces = max(maxNamespaces, len(cs.GetNamespaces()))
		if cs.GetFullClusterAccess() {
			fullClusterAccess++
		}
	}
	eventProps := make(map[string]any)
	eventProps["Total Cluster Scopes"] = len(req.GetClusterScopes())
	eventProps["Cluster Scopes With Full Access"] = fullClusterAccess
	eventProps["Max Namespaces In Scopes"] = maxNamespaces
	for p, a := range req.GetPermissions() {
		eventProps[p] = a.String()
	}
	centralclient.Singleton().Track("OCP Token Issued", eventProps, client)
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
