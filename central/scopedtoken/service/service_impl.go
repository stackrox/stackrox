package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/apitoken/backend"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/accessscope/dynamic"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"google.golang.org/grpc"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
)

const (
	// MaxTokenTTL is the maximum allowed TTL for a scoped token (5 minutes).
	MaxTokenTTL = 5 * time.Minute

	// DefaultTokenTTL is the default TTL if not specified in the request.
	DefaultTokenTTL = 5 * time.Minute
)

var (
	log = logging.LoggerForModule()

	// authorizer ensures only Sensor services can call this API.
	authorizer = idcheck.SensorsOnly()
)

type serviceImpl struct {
	v1.UnimplementedScopedTokenServiceServer

	tokenBackend backend.Backend
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterScopedTokenServiceServer(grpcServer, s)
}

// RegisterServiceHandler is a no-op since this service is only accessible via gRPC (not HTTP).
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	// This service is not exposed via HTTP/REST, only gRPC.
	// Sensor communicates with Central via gRPC only.
	return nil
}

// AuthFuncOverride returns the authorizer for this service.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// IssueToken generates a short-lived, scoped token for a validated user.
//
// The requesting Sensor must:
// 1. Authenticate via mTLS (service certificate) - enforced by authorizer
// 2. Provide a user_identifier from a K8s TokenReview
// 3. Have validated the user's K8s RBAC permissions
// 4. Match cluster_name to its own cluster identity
//
// The issued token will:
// - Have the Analyst role
// - Include a dynamic access scope (cluster/namespace/deployment)
// - Expire in at most 5 minutes
// - Not be stored in the database (ephemeral)
func (s *serviceImpl) IssueToken(ctx context.Context, req *v1.IssueScopedTokenRequest) (*v1.IssueScopedTokenResponse, error) {
	// Note: Authorization (Sensor-only) is handled by AuthFuncOverride.
	// At this point, we know the caller is an authenticated Sensor service.

	// Validate request parameters
	if err := validateRequest(req); err != nil {
		return nil, err
	}

	// Build dynamic scope
	dynamicScope, err := dynamic.BuildDynamicScope(req.GetClusterName(), req.GetNamespace(), req.GetDeployment())
	if err != nil {
		log.Warnw("Invalid scope parameters in token request",
			logging.Err(err),
			logging.String("cluster", req.GetClusterName()),
			logging.String("namespace", req.GetNamespace()),
			logging.String("deployment", req.GetDeployment()),
		)
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	// Determine TTL (max 5 minutes)
	ttl := determineTTL(req.GetTtl())

	// Issue ephemeral scoped token with Analyst role
	roleNames := []string{accesscontrol.Analyst}
	tokenName := generateTokenName(req)

	token, expiresAt, err := s.tokenBackend.IssueEphemeralScopedToken(
		ctx,
		tokenName,
		roleNames,
		dynamicScope,
		ttl,
	)
	if err != nil {
		log.Errorw("Failed to issue scoped token",
			logging.Err(err),
			logging.String("user", req.GetUserIdentifier()),
			logging.String("scope", dynamic.ScopeDescription(dynamicScope)),
		)
		return nil, errors.Errorf("failed to issue token: %v", err)
	}

	log.Infow("Issued scoped token",
		logging.String("user", req.GetUserIdentifier()),
		logging.String("cluster", req.GetClusterName()),
		logging.String("namespace", req.GetNamespace()),
		logging.String("deployment", req.GetDeployment()),
		logging.String("ttl", ttl.String()),
		logging.String("scope_desc", dynamic.ScopeDescription(dynamicScope)),
	)

	return &v1.IssueScopedTokenResponse{
		Token:     token,
		ExpiresAt: protoconv.ConvertTimeToTimestamp(*expiresAt),
	}, nil
}

// validateRequest validates the IssueScopedTokenRequest parameters.
func validateRequest(req *v1.IssueScopedTokenRequest) error {
	if req == nil {
		return errox.InvalidArgs.New("request is nil")
	}

	if req.GetUserIdentifier() == "" {
		return errox.InvalidArgs.New("user_identifier is required")
	}

	if req.GetClusterName() == "" {
		return errox.InvalidArgs.New("cluster_name is required")
	}

	// Note: namespace and deployment are optional (empty means broader scope)
	// The dynamic.BuildDynamicScope function will validate their format if provided

	return nil
}

// determineTTL determines the token TTL based on the request, capped at MaxTokenTTL.
func determineTTL(requestedTTL *durationpb.Duration) time.Duration {
	if requestedTTL == nil {
		return DefaultTokenTTL
	}

	ttl := requestedTTL.AsDuration()
	if ttl <= 0 {
		return DefaultTokenTTL
	}

	if ttl > MaxTokenTTL {
		return MaxTokenTTL
	}

	return ttl
}

// generateTokenName creates a descriptive name for the token for logging/debugging.
func generateTokenName(req *v1.IssueScopedTokenRequest) string {
	// Example: "ocp-console:user@example.com@prod-cluster"
	return "ocp-console:" + req.GetUserIdentifier() + "@" + req.GetClusterName()
}
