package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/graphqlgateway/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockK8sValidator is a mock implementation of K8sValidator for testing
type mockK8sValidator struct {
	validateFunc func(ctx context.Context, bearerToken, namespace, deployment string) (*K8sUserInfo, error)
}

func (m *mockK8sValidator) ValidateDeploymentAccess(ctx context.Context, bearerToken, namespace, deployment string) (*K8sUserInfo, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, bearerToken, namespace, deployment)
	}
	return &K8sUserInfo{Username: "test-user", Groups: []string{"system:authenticated"}}, nil
}

// mockTokenClient is a mock implementation of TokenClient for testing
type mockTokenClient struct {
	requestFunc func(ctx context.Context, username, namespace, deployment string) (*TokenResponse, error)
}

func (m *mockTokenClient) RequestToken(ctx context.Context, username, namespace, deployment string) (*TokenResponse, error) {
	if m.requestFunc != nil {
		return m.requestFunc(ctx, username, namespace, deployment)
	}
	return &TokenResponse{
		Token:     "mock-token-123",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}, nil
}

// mockTokenCache is a mock implementation of cache.TokenCache for testing
type mockTokenCache struct {
	storage map[string]string
	getFunc func(ctx context.Context, key cache.CacheKey) (string, bool)
	setFunc func(ctx context.Context, key cache.CacheKey, token string, ttl time.Duration)
}

func newMockTokenCache() *mockTokenCache {
	return &mockTokenCache{
		storage: make(map[string]string),
	}
}

func (m *mockTokenCache) Get(ctx context.Context, key cache.CacheKey) (string, bool) {
	if m.getFunc != nil {
		return m.getFunc(ctx, key)
	}
	token, found := m.storage[key.String()]
	return token, found
}

func (m *mockTokenCache) Set(ctx context.Context, key cache.CacheKey, token string, ttl time.Duration) {
	if m.setFunc != nil {
		m.setFunc(ctx, key, token, ttl)
	}
	m.storage[key.String()] = token
}

func (m *mockTokenCache) Invalidate(key cache.CacheKey) {
	delete(m.storage, key.String())
}

func (m *mockTokenCache) Clear() {
	m.storage = make(map[string]string)
}

func (m *mockTokenCache) Size() int {
	return len(m.storage)
}

func TestTokenManager_GetToken_CacheHit(t *testing.T) {
	ctx := context.Background()
	mockCache := newMockTokenCache()
	mockValidator := &mockK8sValidator{}
	mockClient := &mockTokenClient{}

	// Pre-populate cache
	cacheKey := cache.NewCacheKey("test-user", "default", "nginx")
	mockCache.Set(ctx, cacheKey, "cached-token-abc", 5*time.Minute)

	manager := NewTokenManager(
		mockValidator,
		mockClient,
		mockCache,
		nil, // centralSignal not needed for cache hit
	)

	token, err := manager.GetToken(ctx, "bearer-token", "default", "nginx")

	require.NoError(t, err)
	assert.Equal(t, "cached-token-abc", token)
}

func TestTokenManager_GetToken_CacheMiss_SuccessfulAcquisition(t *testing.T) {
	ctx := context.Background()
	mockCache := newMockTokenCache()
	mockValidator := &mockK8sValidator{}
	mockClient := &mockTokenClient{
		requestFunc: func(ctx context.Context, username, namespace, deployment string) (*TokenResponse, error) {
			assert.Equal(t, "test-user", username)
			assert.Equal(t, "production", namespace)
			assert.Equal(t, "api-server", deployment)
			return &TokenResponse{
				Token:     "new-token-xyz",
				ExpiresAt: time.Now().Add(5 * time.Minute),
			}, nil
		},
	}

	signal := concurrency.NewErrorSignal()
	signal.Signal() // Signal is ready (Central is reachable)

	manager := NewTokenManager(
		mockValidator,
		mockClient,
		mockCache,
		&signal,
	)

	token, err := manager.GetToken(ctx, "bearer-token", "production", "api-server")

	require.NoError(t, err)
	assert.Equal(t, "new-token-xyz", token)

	// Verify token was cached
	cacheKey := cache.NewCacheKey("test-user", "production", "api-server")
	cachedToken, found := mockCache.Get(ctx, cacheKey)
	assert.True(t, found)
	assert.Equal(t, "new-token-xyz", cachedToken)
}

func TestTokenManager_GetToken_RBACValidationFails(t *testing.T) {
	ctx := context.Background()
	mockCache := newMockTokenCache()
	mockValidator := &mockK8sValidator{
		validateFunc: func(ctx context.Context, bearerToken, namespace, deployment string) (*K8sUserInfo, error) {
			return nil, errors.New("not authorized to access deployment")
		},
	}
	mockClient := &mockTokenClient{}

	manager := NewTokenManager(
		mockValidator,
		mockClient,
		mockCache,
		nil,
	)

	token, err := manager.GetToken(ctx, "invalid-token", "default", "nginx")

	require.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "not authorized")
}

func TestTokenManager_GetToken_CentralOffline_NoCachedToken(t *testing.T) {
	ctx := context.Background()
	mockCache := newMockTokenCache()
	mockValidator := &mockK8sValidator{}
	mockClient := &mockTokenClient{}

	signal := concurrency.NewErrorSignal()
	signal.SignalWithError(errors.New("connection to Central lost"))

	manager := NewTokenManager(
		mockValidator,
		mockClient,
		mockCache,
		&signal,
	)

	token, err := manager.GetToken(ctx, "bearer-token", "default", "nginx")

	require.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "offline")
}

func TestTokenManager_GetToken_CentralOffline_WithCachedToken(t *testing.T) {
	ctx := context.Background()
	mockCache := newMockTokenCache()
	mockValidator := &mockK8sValidator{}
	mockClient := &mockTokenClient{}

	// Pre-populate cache
	cacheKey := cache.NewCacheKey("test-user", "default", "nginx")
	mockCache.Set(ctx, cacheKey, "cached-token-xyz", 5*time.Minute)

	signal := concurrency.NewErrorSignal()
	signal.SignalWithError(errors.New("connection to Central lost"))

	manager := NewTokenManager(
		mockValidator,
		mockClient,
		mockCache,
		&signal,
	)

	// Should return cached token even though Central is offline
	token, err := manager.GetToken(ctx, "bearer-token", "default", "nginx")

	require.NoError(t, err)
	assert.Equal(t, "cached-token-xyz", token)
}

func TestTokenManager_GetToken_TokenClientFails(t *testing.T) {
	ctx := context.Background()
	mockCache := newMockTokenCache()
	mockValidator := &mockK8sValidator{}
	mockClient := &mockTokenClient{
		requestFunc: func(ctx context.Context, username, namespace, deployment string) (*TokenResponse, error) {
			return nil, errors.New("gRPC connection error")
		},
	}

	signal := concurrency.NewErrorSignal()
	signal.Signal() // Central is reachable

	manager := NewTokenManager(
		mockValidator,
		mockClient,
		mockCache,
		&signal,
	)

	token, err := manager.GetToken(ctx, "bearer-token", "default", "nginx")

	require.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "failed to acquire scoped token")
}

func TestTokenManager_GetToken_ExpiredToken_NotCached(t *testing.T) {
	ctx := context.Background()
	mockCache := newMockTokenCache()
	mockValidator := &mockK8sValidator{}

	var cacheSetCalled bool
	mockClient := &mockTokenClient{
		requestFunc: func(ctx context.Context, username, namespace, deployment string) (*TokenResponse, error) {
			// Return already-expired token
			return &TokenResponse{
				Token:     "expired-token",
				ExpiresAt: time.Now().Add(-1 * time.Minute), // Already expired
			}, nil
		},
	}

	mockCache.setFunc = func(ctx context.Context, key cache.CacheKey, token string, ttl time.Duration) {
		cacheSetCalled = true
	}

	signal := concurrency.NewErrorSignal()
	signal.Signal()

	manager := NewTokenManager(
		mockValidator,
		mockClient,
		mockCache,
		&signal,
	)

	token, err := manager.GetToken(ctx, "bearer-token", "default", "nginx")

	require.NoError(t, err)
	assert.Equal(t, "expired-token", token)

	// Verify token was NOT cached (because TTL was negative)
	assert.False(t, cacheSetCalled, "expired token should not be cached")
}

func TestTokenManager_GetToken_NilCentralSignal(t *testing.T) {
	ctx := context.Background()
	mockCache := newMockTokenCache()
	mockValidator := &mockK8sValidator{}
	mockClient := &mockTokenClient{
		requestFunc: func(ctx context.Context, username, namespace, deployment string) (*TokenResponse, error) {
			return &TokenResponse{
				Token:     "token-no-signal",
				ExpiresAt: time.Now().Add(5 * time.Minute),
			}, nil
		},
	}

	manager := NewTokenManager(
		mockValidator,
		mockClient,
		mockCache,
		nil, // centralSignal is nil
	)

	// Should work fine with nil signal
	token, err := manager.GetToken(ctx, "bearer-token", "default", "nginx")

	require.NoError(t, err)
	assert.Equal(t, "token-no-signal", token)
}

func TestTokenManager_GetToken_CacheKeyGeneration(t *testing.T) {
	ctx := context.Background()
	mockCache := newMockTokenCache()
	mockValidator := &mockK8sValidator{
		validateFunc: func(ctx context.Context, bearerToken, namespace, deployment string) (*K8sUserInfo, error) {
			return &K8sUserInfo{
				Username: "custom-user",
				Groups:   []string{"developers"},
			}, nil
		},
	}
	mockClient := &mockTokenClient{}

	var capturedCacheKey cache.CacheKey
	mockCache.getFunc = func(ctx context.Context, key cache.CacheKey) (string, bool) {
		capturedCacheKey = key
		return "", false // Cache miss
	}

	signal := concurrency.NewErrorSignal()
	signal.Signal()

	manager := NewTokenManager(
		mockValidator,
		mockClient,
		mockCache,
		&signal,
	)

	_, _ = manager.GetToken(ctx, "bearer-token", "staging", "database")

	// Verify cache key was generated correctly
	expectedKey := cache.NewCacheKey("custom-user", "staging", "database")
	assert.Equal(t, expectedKey.String(), capturedCacheKey.String())
}
