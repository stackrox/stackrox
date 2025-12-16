package auth

import (
	"context"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"google.golang.org/grpc"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
)

const (
	// TokenRequestTimeout is the timeout for token requests to Central
	TokenRequestTimeout = 10 * time.Second

	// DefaultTokenTTL is the requested TTL for tokens (will be capped by Central at 5 minutes)
	DefaultTokenTTL = 5 * time.Minute
)

// TokenResponse contains the token and its expiration time.
type TokenResponse struct {
	Token     string
	ExpiresAt time.Time
}

// TokenClient handles communication with Central's ScopedTokenService.
type TokenClient struct {
	centralConn *grpc.ClientConn
	clusterName string
}

// NewTokenClient creates a new token client for requesting scoped tokens from Central.
func NewTokenClient(centralConn *grpc.ClientConn, clusterName string) *TokenClient {
	return &TokenClient{
		centralConn: centralConn,
		clusterName: clusterName,
	}
}

// RequestToken requests a scoped token from Central for the given user and scope.
//
// Parameters:
// - userID: The Kubernetes user identifier (from TokenReview)
// - namespace: The namespace to scope the token to (empty = all namespaces)
// - deployment: The deployment to scope the token to (empty = all deployments)
//
// Returns the token and its expiration time, or an error if the request fails.
func (c *TokenClient) RequestToken(ctx context.Context, userID, namespace, deployment string) (*TokenResponse, error) {
	client := v1.NewScopedTokenServiceClient(c.centralConn)

	req := &v1.IssueScopedTokenRequest{
		UserIdentifier: userID,
		ClusterName:    c.clusterName,
		Namespace:      namespace,
		Deployment:     deployment,
		Ttl:            durationpb.New(DefaultTokenTTL),
	}

	// Create context with timeout
	reqCtx, cancel := context.WithTimeout(ctx, TokenRequestTimeout)
	defer cancel()

	// Request token from Central
	resp, err := client.IssueToken(reqCtx, req)
	if err != nil {
		log.Errorw("Failed to request scoped token from Central",
			logging.Err(err),
			logging.String("user", userID),
			logging.String("namespace", namespace),
			logging.String("deployment", deployment),
		)
		return nil, errors.Wrap(err, "failed to request token from Central")
	}

	if resp.GetToken() == "" {
		return nil, errox.ServerError.New("Central returned empty token")
	}

	expiresAt := protoconv.ConvertTimestampToTimeOrNow(resp.GetExpiresAt())

	log.Infow("Received scoped token from Central",
		logging.String("user", userID),
		logging.String("namespace", namespace),
		logging.String("deployment", deployment),
		logging.String("expires_at", expiresAt.String()),
	)

	return &TokenResponse{
		Token:     resp.GetToken(),
		ExpiresAt: expiresAt,
	}, nil
}
