package centralproxy

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// fakeClusterIDGetter is a test implementation of clusterIDGetter.
type fakeClusterIDGetter struct {
	clusterID string
}

func (f *fakeClusterIDGetter) GetNoWait() string {
	return f.clusterID
}

// fakeTokenServiceClient is a test implementation of centralv1.TokenServiceClient.
type fakeTokenServiceClient struct {
	response *centralv1.GenerateTokenForPermissionsAndScopeResponse
	err      error

	// Capture the request for verification
	lastRequest *centralv1.GenerateTokenForPermissionsAndScopeRequest
}

func (f *fakeTokenServiceClient) GenerateTokenForPermissionsAndScope(
	ctx context.Context,
	in *centralv1.GenerateTokenForPermissionsAndScopeRequest,
	opts ...grpc.CallOption,
) (*centralv1.GenerateTokenForPermissionsAndScopeResponse, error) {
	f.lastRequest = in
	return f.response, f.err
}

func TestScopedTokenTransport_RoundTrip(t *testing.T) {
	tests := []struct {
		name           string
		namespaceScope string
		expectedToken  string
	}{
		{
			name:           "empty scope",
			namespaceScope: "",
			expectedToken:  "token-for-empty-scope",
		},
		{
			name:           "specific namespace",
			namespaceScope: "my-namespace",
			expectedToken:  "token-for-namespace",
		},
		{
			name:           "cluster-wide access",
			namespaceScope: FullClusterAccessScope,
			expectedToken:  "token-for-cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedAuthHeader string
			mockBase := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				capturedAuthHeader = req.Header.Get("Authorization")
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			})

			fakeClient := &fakeTokenServiceClient{
				response: &centralv1.GenerateTokenForPermissionsAndScopeResponse{
					Token: tt.expectedToken,
				},
			}

			tp := &tokenProvider{
				client:          fakeClient,
				clusterIDGetter: &fakeClusterIDGetter{clusterID: "test-cluster-id"},
				tokenCache:      expiringcache.NewExpiringCache[string, string](tokenCacheTTL),
			}

			transport := &scopedTokenTransport{
				base:          mockBase,
				tokenProvider: tp,
			}

			req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
			if tt.namespaceScope != "" {
				req.Header.Set(stackroxNamespaceHeader, tt.namespaceScope)
			}

			resp, err := transport.RoundTrip(req)
			require.NoError(t, err)
			require.NotNil(t, resp)

			assert.Equal(t, "Bearer "+tt.expectedToken, capturedAuthHeader)
		})
	}
}

func TestScopedTokenTransport_RoundTrip_Error(t *testing.T) {
	t.Run("token provider error propagates", func(t *testing.T) {
		mockBase := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			t.Fatal("base transport should not be called")
			return nil, nil
		})

		// Token provider with no client set - will return error
		tp := &tokenProvider{
			client:          nil, // Not initialized
			clusterIDGetter: &fakeClusterIDGetter{clusterID: "test-cluster-id"},
			tokenCache:      expiringcache.NewExpiringCache[string, string](tokenCacheTTL),
		}

		transport := &scopedTokenTransport{
			base:          mockBase,
			tokenProvider: tp,
		}

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		resp, err := transport.RoundTrip(req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("base transport error propagates", func(t *testing.T) {
		baseErr := errors.New("connection refused")
		mockBase := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			// Verify the auth header was set before the error
			assert.NotEmpty(t, req.Header.Get("Authorization"))
			return nil, baseErr
		})

		fakeClient := &fakeTokenServiceClient{
			response: &centralv1.GenerateTokenForPermissionsAndScopeResponse{
				Token: "test-token",
			},
		}

		tp := &tokenProvider{
			client:          fakeClient,
			clusterIDGetter: &fakeClusterIDGetter{clusterID: "test-cluster-id"},
			tokenCache:      expiringcache.NewExpiringCache[string, string](tokenCacheTTL),
		}

		transport := &scopedTokenTransport{
			base:          mockBase,
			tokenProvider: tp,
		}

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		resp, err := transport.RoundTrip(req)

		assert.ErrorIs(t, err, baseErr)
		assert.Nil(t, resp)
	})
}

func TestTokenProvider_GetTokenForScope(t *testing.T) {
	tests := []struct {
		name           string
		namespaceScope string
		clusterID      string
		wantScopes     bool
		wantFullAccess bool
		wantNamespaces []string
	}{
		{
			name:           "empty scope - auth only",
			namespaceScope: "",
			clusterID:      "test-cluster-id",
			wantScopes:     false,
		},
		{
			name:           "specific namespace",
			namespaceScope: "my-namespace",
			clusterID:      "test-cluster-id",
			wantScopes:     true,
			wantNamespaces: []string{"my-namespace"},
		},
		{
			name:           "cluster-wide access",
			namespaceScope: FullClusterAccessScope,
			clusterID:      "test-cluster-id",
			wantScopes:     true,
			wantFullAccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := &fakeTokenServiceClient{
				response: &centralv1.GenerateTokenForPermissionsAndScopeResponse{
					Token: "test-token-123",
				},
			}

			provider := &tokenProvider{
				client:          fakeClient,
				clusterIDGetter: &fakeClusterIDGetter{clusterID: tt.clusterID},
				tokenCache:      expiringcache.NewExpiringCache[string, string](tokenCacheTTL),
			}

			token, err := provider.getTokenForScope(context.Background(), tt.namespaceScope)
			require.NoError(t, err)
			assert.Equal(t, "test-token-123", token)

			// Verify request
			req := fakeClient.lastRequest
			require.NotNil(t, req)

			// Verify permissions
			assert.Equal(t, centralv1.Access_READ_ACCESS, req.GetPermissions()[permissionImage])
			assert.Equal(t, centralv1.Access_READ_ACCESS, req.GetPermissions()[permissionDeployment])

			// Verify scopes
			if tt.wantScopes {
				require.Len(t, req.GetClusterScopes(), 1)
				scope := req.GetClusterScopes()[0]
				assert.Equal(t, tt.clusterID, scope.GetClusterId())

				if tt.wantFullAccess {
					assert.True(t, scope.GetFullClusterAccess())
				} else {
					assert.False(t, scope.GetFullClusterAccess())
					assert.Equal(t, tt.wantNamespaces, scope.GetNamespaces())
				}
			} else {
				assert.Empty(t, req.GetClusterScopes())
			}
		})
	}
}

func TestTokenProvider_Caching(t *testing.T) {
	t.Run("tokens are cached", func(t *testing.T) {
		callCount := 0
		fakeClient := &dynamicFakeTokenServiceClient{
			getToken: func() string {
				callCount++
				return "cached-token"
			},
		}

		provider := &tokenProvider{
			client:          fakeClient,
			clusterIDGetter: &fakeClusterIDGetter{clusterID: "test-cluster-id"},
			tokenCache:      expiringcache.NewExpiringCache[string, string](tokenCacheTTL),
		}

		// First call should hit the API
		token1, err := provider.getTokenForScope(context.Background(), "namespace-a")
		require.NoError(t, err)
		assert.Equal(t, "cached-token", token1)
		assert.Equal(t, 1, callCount)

		// Second call with same scope should use cache
		token2, err := provider.getTokenForScope(context.Background(), "namespace-a")
		require.NoError(t, err)
		assert.Equal(t, "cached-token", token2)
		assert.Equal(t, 1, callCount, "should use cached token")

		// Third call with different scope should hit API again
		token3, err := provider.getTokenForScope(context.Background(), "namespace-b")
		require.NoError(t, err)
		assert.Equal(t, "cached-token", token3)
		assert.Equal(t, 2, callCount, "different scope should request new token")
	})

	t.Run("different scopes get different cache entries", func(t *testing.T) {
		tokenIndex := 0
		tokens := []string{"token-1", "token-2", "token-3"}

		fakeClient := &dynamicFakeTokenServiceClient{
			getToken: func() string {
				token := tokens[tokenIndex]
				tokenIndex++
				return token
			},
		}

		provider := &tokenProvider{
			client:          fakeClient,
			clusterIDGetter: &fakeClusterIDGetter{clusterID: "test-cluster-id"},
			tokenCache:      expiringcache.NewExpiringCache[string, string](tokenCacheTTL),
		}

		// Request for scope A
		tokenA, err := provider.getTokenForScope(context.Background(), "scope-a")
		require.NoError(t, err)
		assert.Equal(t, "token-1", tokenA)

		// Request for scope B (different scope, should get new token)
		tokenB, err := provider.getTokenForScope(context.Background(), "scope-b")
		require.NoError(t, err)
		assert.Equal(t, "token-2", tokenB)

		// Request for scope A again (should use cached)
		tokenACached, err := provider.getTokenForScope(context.Background(), "scope-a")
		require.NoError(t, err)
		assert.Equal(t, "token-1", tokenACached)

		// Request for scope B again (should use cached)
		tokenBCached, err := provider.getTokenForScope(context.Background(), "scope-b")
		require.NoError(t, err)
		assert.Equal(t, "token-2", tokenBCached)
	})
}

func TestTokenProvider_ErrorHandling(t *testing.T) {
	t.Run("no client returns error", func(t *testing.T) {
		provider := &tokenProvider{
			client:          nil,
			clusterIDGetter: &fakeClusterIDGetter{clusterID: "test-cluster-id"},
			tokenCache:      expiringcache.NewExpiringCache[string, string](tokenCacheTTL),
		}

		_, err := provider.getTokenForScope(context.Background(), "namespace")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("empty token response returns error", func(t *testing.T) {
		fakeClient := &fakeTokenServiceClient{
			response: &centralv1.GenerateTokenForPermissionsAndScopeResponse{
				Token: "", // Empty token
			},
		}

		provider := &tokenProvider{
			client:          fakeClient,
			clusterIDGetter: &fakeClusterIDGetter{clusterID: "test-cluster-id"},
			tokenCache:      expiringcache.NewExpiringCache[string, string](tokenCacheTTL),
		}

		_, err := provider.getTokenForScope(context.Background(), "namespace")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty token")

		// Ensure nothing was cached for this scope
		_, found := provider.tokenCache.Get("namespace")
		assert.False(t, found)
	})

	t.Run("client error is returned and not cached", func(t *testing.T) {
		fakeErr := errors.New("rpc failure")

		fakeClient := &fakeTokenServiceClient{
			err: fakeErr,
		}

		provider := &tokenProvider{
			client:          fakeClient,
			clusterIDGetter: &fakeClusterIDGetter{clusterID: "test-cluster-id"},
			tokenCache:      expiringcache.NewExpiringCache[string, string](tokenCacheTTL),
		}

		_, err := provider.getTokenForScope(context.Background(), "namespace-error")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rpc failure")

		// Ensure token is not cached for the failing scope
		_, found := provider.tokenCache.Get("namespace-error")
		assert.False(t, found)
	})
}

func TestBuildTokenRequest(t *testing.T) {
	provider := &tokenProvider{
		clusterIDGetter: &fakeClusterIDGetter{clusterID: "my-cluster-id"},
	}

	t.Run("empty scope", func(t *testing.T) {
		req, err := provider.buildTokenRequest("")
		require.NoError(t, err)
		assert.Equal(t, centralv1.Access_READ_ACCESS, req.GetPermissions()[permissionImage])
		assert.Equal(t, centralv1.Access_READ_ACCESS, req.GetPermissions()[permissionDeployment])
		assert.Empty(t, req.GetClusterScopes())
		assert.NotNil(t, req.GetLifetime())
	})

	t.Run("specific namespace", func(t *testing.T) {
		req, err := provider.buildTokenRequest("prod")
		require.NoError(t, err)
		require.Len(t, req.GetClusterScopes(), 1)
		assert.Equal(t, "my-cluster-id", req.GetClusterScopes()[0].GetClusterId())
		assert.False(t, req.GetClusterScopes()[0].GetFullClusterAccess())
		assert.Equal(t, []string{"prod"}, req.GetClusterScopes()[0].GetNamespaces())
	})

	t.Run("cluster-wide scope", func(t *testing.T) {
		req, err := provider.buildTokenRequest(FullClusterAccessScope)
		require.NoError(t, err)
		require.Len(t, req.GetClusterScopes(), 1)
		assert.Equal(t, "my-cluster-id", req.GetClusterScopes()[0].GetClusterId())
		assert.True(t, req.GetClusterScopes()[0].GetFullClusterAccess())
		assert.Empty(t, req.GetClusterScopes()[0].GetNamespaces())
	})

	t.Run("empty cluster ID returns error", func(t *testing.T) {
		emptyProvider := &tokenProvider{
			clusterIDGetter: &fakeClusterIDGetter{clusterID: ""},
		}
		req, err := emptyProvider.buildTokenRequest("namespace")
		require.Error(t, err)
		assert.Nil(t, req)
		assert.Contains(t, err.Error(), "cluster ID not available")
	})
}

// dynamicFakeTokenServiceClient allows dynamic token generation for testing.
type dynamicFakeTokenServiceClient struct {
	getToken func() string
}

func (d *dynamicFakeTokenServiceClient) GenerateTokenForPermissionsAndScope(
	ctx context.Context,
	in *centralv1.GenerateTokenForPermissionsAndScopeRequest,
	opts ...grpc.CallOption,
) (*centralv1.GenerateTokenForPermissionsAndScopeResponse, error) {
	return &centralv1.GenerateTokenForPermissionsAndScopeResponse{
		Token: d.getToken(),
	}, nil
}

// roundTripperFunc is a helper to create RoundTripper from a function.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
