package auth

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/graphqlgateway/cache"
)

// K8sValidatorInterface defines the interface for validating Kubernetes RBAC.
type K8sValidatorInterface interface {
	ValidateDeploymentAccess(ctx context.Context, token, namespace, deployment string) (*K8sUserInfo, error)
}

// TokenClientInterface defines the interface for requesting scoped tokens from Central.
type TokenClientInterface interface {
	RequestToken(ctx context.Context, userID, namespace, deployment string) (*TokenResponse, error)
}

// TokenManager orchestrates token acquisition flow:
// 1. Validates K8s RBAC (SubjectAccessReview)
// 2. Checks cache for existing token
// 3. Requests new token from Central if cache miss
// 4. Caches the new token
type TokenManager struct {
	k8sValidator  K8sValidatorInterface
	tokenClient   TokenClientInterface
	tokenCache    cache.TokenCache
	centralSignal concurrency.ReadOnlyErrorSignal
}

// NewTokenManager creates a new token manager.
//
// Parameters:
// - k8sValidator: Validates K8s RBAC permissions
// - tokenClient: Communicates with Central's ScopedTokenService
// - tokenCache: Caches tokens to reduce Central load
// - centralSignal: Signal indicating Central connectivity status
func NewTokenManager(
	k8sValidator K8sValidatorInterface,
	tokenClient TokenClientInterface,
	tokenCache cache.TokenCache,
	centralSignal concurrency.ReadOnlyErrorSignal,
) *TokenManager {
	return &TokenManager{
		k8sValidator:  k8sValidator,
		tokenClient:   tokenClient,
		tokenCache:    tokenCache,
		centralSignal: centralSignal,
	}
}

// GetToken orchestrates the complete token acquisition flow.
//
// Steps:
// 1. Validate K8s RBAC via SubjectAccessReview
// 2. Check cache for existing token
// 3. If cache miss, request new token from Central
// 4. Cache the new token
// 5. Return the token
//
// If Central is offline, attempts to use cached token. If no cached token
// available, returns ServiceUnavailable error.
func (m *TokenManager) GetToken(ctx context.Context, bearerToken, namespace, deployment string) (string, error) {
	// Step 1: Validate K8s RBAC permissions
	userInfo, err := m.k8sValidator.ValidateDeploymentAccess(ctx, bearerToken, namespace, deployment)
	if err != nil {
		// This returns PermissionDenied or Unauthenticated error
		return "", err
	}

	// Step 2: Check cache
	cacheKey := cache.NewCacheKey(userInfo.Username, namespace, deployment)
	if cachedToken, found := m.tokenCache.Get(ctx, cacheKey); found {
		log.Debugw("Token cache hit",
			logging.String("user", userInfo.Username),
			logging.String("key", cacheKey.String()),
		)
		return cachedToken, nil
	}

	log.Debugw("Token cache miss",
		logging.String("user", userInfo.Username),
		logging.String("key", cacheKey.String()),
	)

	// Step 3: Request new token from Central
	// Check if Central is reachable (if signal is available)
	if m.centralSignal != nil {
		if err := m.centralSignal.Err(); err != nil {
			// Central is offline
			log.Warnw("Central is offline, cannot request new token",
				logging.Err(err),
				logging.String("user", userInfo.Username),
			)
			// We already checked cache above, so no cached token available
			return "", errox.ServerError.New("Central is offline and no cached token available")
		}
	}

	tokenResp, err := m.tokenClient.RequestToken(ctx, userInfo.Username, namespace, deployment)
	if err != nil {
		// Token request failed - could be network error, Central error, etc.
		log.Errorw("Failed to request token from Central",
			logging.Err(err),
			logging.String("user", userInfo.Username),
		)
		return "", errors.Wrap(err, "failed to acquire scoped token")
	}

	// Step 4: Cache the new token
	ttl := tokenResp.ExpiresAt.Sub(time.Now())
	if ttl > 0 {
		m.tokenCache.Set(ctx, cacheKey, tokenResp.Token, ttl)
		log.Debugw("Cached new token",
			logging.String("user", userInfo.Username),
			logging.String("ttl", ttl.String()),
		)
	} else {
		log.Warnw("Token already expired, not caching",
			logging.String("user", userInfo.Username),
			logging.String("expires_at", tokenResp.ExpiresAt.String()),
		)
	}

	// Step 5: Return the token
	return tokenResp.Token, nil
}
