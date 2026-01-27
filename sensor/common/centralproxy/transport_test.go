package centralproxy

import (
	"context"
	"errors"
	"fmt"
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

// newTestTokenProvider creates a tokenProvider for testing with the given client.
func newTestTokenProvider(client centralv1.TokenServiceClient, clusterID string) *tokenProvider {
	tp := &tokenProvider{
		clusterIDGetter: &fakeClusterIDGetter{clusterID: clusterID},
		tokenCache:      expiringcache.NewExpiringCache[string, string](tokenCacheTTL),
	}
	if client != nil {
		tp.client.Store(&client)
	}
	return tp
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

			transport := &scopedTokenTransport{
				base:          mockBase,
				tokenProvider: newTestTokenProvider(fakeClient, "test-cluster-id"),
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
		transport := &scopedTokenTransport{
			base:          mockBase,
			tokenProvider: newTestTokenProvider(nil, "test-cluster-id"),
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

		transport := &scopedTokenTransport{
			base:          mockBase,
			tokenProvider: newTestTokenProvider(fakeClient, "test-cluster-id"),
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

			provider := newTestTokenProvider(fakeClient, tt.clusterID)

			token, err := provider.getTokenForScope(context.Background(), tt.namespaceScope)
			require.NoError(t, err)
			assert.Equal(t, "test-token-123", token)

			// Verify request
			req := fakeClient.lastRequest
			require.NotNil(t, req)

			// Verify permissions
			assert.Equal(t, centralv1.Access_READ_ACCESS, req.GetPermissions()["Image"])
			assert.Equal(t, centralv1.Access_READ_ACCESS, req.GetPermissions()["Deployment"])

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

		provider := newTestTokenProvider(fakeClient, "test-cluster-id")

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

		provider := newTestTokenProvider(fakeClient, "test-cluster-id")

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
		provider := newTestTokenProvider(nil, "test-cluster-id")

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

		provider := newTestTokenProvider(fakeClient, "test-cluster-id")

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

		provider := newTestTokenProvider(fakeClient, "test-cluster-id")

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
		assert.Equal(t, centralv1.Access_READ_ACCESS, req.GetPermissions()["Image"])
		assert.Equal(t, centralv1.Access_READ_ACCESS, req.GetPermissions()["Deployment"])
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

func TestScopedTokenTransport_InvalidateOnUnauthorized(t *testing.T) {
	t.Run("401 response triggers retry with fresh token", func(t *testing.T) {
		tokenCallCount := 0
		fakeClient := &dynamicFakeTokenServiceClient{
			getToken: func() string {
				tokenCallCount++
				return fmt.Sprintf("token-%d", tokenCallCount)
			},
		}

		requestCount := 0
		mockBase := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			requestCount++
			// First request returns 401, retry returns 200
			if req.Header.Get("Authorization") == "Bearer token-1" {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error":"unauthorized"}`)),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		transport := &scopedTokenTransport{
			base:          mockBase,
			tokenProvider: newTestTokenProvider(fakeClient, "test-cluster-id"),
		}

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		req.Header.Set(stackroxNamespaceHeader, "test-namespace")

		// Single RoundTrip should retry internally and return success
		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "should return success after retry")
		assert.Equal(t, 2, tokenCallCount, "should have requested two tokens (original + retry)")
		assert.Equal(t, 2, requestCount, "should have made two requests (original + retry)")
	})

	t.Run("403 response triggers retry with fresh token", func(t *testing.T) {
		tokenCallCount := 0
		fakeClient := &dynamicFakeTokenServiceClient{
			getToken: func() string {
				tokenCallCount++
				return fmt.Sprintf("token-%d", tokenCallCount)
			},
		}

		requestCount := 0
		mockBase := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			requestCount++
			// First request returns 403, retry returns 200
			if req.Header.Get("Authorization") == "Bearer token-1" {
				return &http.Response{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"error":"forbidden"}`)),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		transport := &scopedTokenTransport{
			base:          mockBase,
			tokenProvider: newTestTokenProvider(fakeClient, "test-cluster-id"),
		}

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		req.Header.Set(stackroxNamespaceHeader, "test-namespace")

		// Single RoundTrip should retry internally and return success
		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "should return success after retry")
		assert.Equal(t, 2, tokenCallCount, "should have requested two tokens (original + retry)")
		assert.Equal(t, 2, requestCount, "should have made two requests (original + retry)")
	})

	t.Run("retry only happens once", func(t *testing.T) {
		tokenCallCount := 0
		fakeClient := &dynamicFakeTokenServiceClient{
			getToken: func() string {
				tokenCallCount++
				return fmt.Sprintf("token-%d", tokenCallCount)
			},
		}

		requestCount := 0
		mockBase := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			requestCount++
			// Always return 401
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"error":"unauthorized"}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		transport := &scopedTokenTransport{
			base:          mockBase,
			tokenProvider: newTestTokenProvider(fakeClient, "test-cluster-id"),
		}

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		req.Header.Set(stackroxNamespaceHeader, "test-namespace")

		// Should retry once and then return the 401
		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "should return 401 after retry fails")
		assert.Equal(t, 2, tokenCallCount, "should have requested exactly two tokens")
		assert.Equal(t, 2, requestCount, "should have made exactly two requests")
	})

	t.Run("other error responses do not invalidate cache", func(t *testing.T) {
		statusCodes := []int{
			http.StatusOK,
			http.StatusNotFound,
			http.StatusInternalServerError,
		}

		for _, statusCode := range statusCodes {
			t.Run(fmt.Sprintf("status %d", statusCode), func(t *testing.T) {
				callCount := 0
				fakeClient := &dynamicFakeTokenServiceClient{
					getToken: func() string {
						callCount++
						return "cached-token"
					},
				}

				mockBase := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(strings.NewReader(`{}`)),
						Header:     http.Header{"Content-Type": []string{"application/json"}},
					}, nil
				})

				transport := &scopedTokenTransport{
					base:          mockBase,
					tokenProvider: newTestTokenProvider(fakeClient, "test-cluster-id"),
				}

				req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
				req.Header.Set(stackroxNamespaceHeader, "test-namespace")

				// First request
				_, err := transport.RoundTrip(req)
				require.NoError(t, err)
				assert.Equal(t, 1, callCount)

				// Second request - should use cached token
				_, err = transport.RoundTrip(req)
				require.NoError(t, err)
				assert.Equal(t, 1, callCount, "token should still be cached for status %d", statusCode)
			})
		}
	})

	t.Run("transport error does not invalidate cache", func(t *testing.T) {
		callCount := 0
		fakeClient := &dynamicFakeTokenServiceClient{
			getToken: func() string {
				callCount++
				return "cached-token"
			},
		}

		transportErr := errors.New("connection refused")
		mockBase := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return nil, transportErr
		})

		transport := &scopedTokenTransport{
			base:          mockBase,
			tokenProvider: newTestTokenProvider(fakeClient, "test-cluster-id"),
		}

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		req.Header.Set(stackroxNamespaceHeader, "test-namespace")

		// First request - transport error
		_, err := transport.RoundTrip(req)
		require.Error(t, err)
		assert.Equal(t, 1, callCount)

		// Second request - should use cached token (error didn't invalidate)
		_, err = transport.RoundTrip(req)
		require.Error(t, err)
		assert.Equal(t, 1, callCount, "token should still be cached after transport error")
	})
}

func TestTokenProvider_InvalidateToken(t *testing.T) {
	t.Run("invalidateToken removes token from cache", func(t *testing.T) {
		callCount := 0
		fakeClient := &dynamicFakeTokenServiceClient{
			getToken: func() string {
				callCount++
				return fmt.Sprintf("token-%d", callCount)
			},
		}

		provider := newTestTokenProvider(fakeClient, "test-cluster-id")

		// Get token (causes cache)
		token1, err := provider.getTokenForScope(context.Background(), "my-scope")
		require.NoError(t, err)
		assert.Equal(t, "token-1", token1)
		assert.Equal(t, 1, callCount)

		// Get again - should be cached
		token2, err := provider.getTokenForScope(context.Background(), "my-scope")
		require.NoError(t, err)
		assert.Equal(t, "token-1", token2)
		assert.Equal(t, 1, callCount)

		// Invalidate
		provider.invalidateToken("my-scope")

		// Get again - should fetch new token
		token3, err := provider.getTokenForScope(context.Background(), "my-scope")
		require.NoError(t, err)
		assert.Equal(t, "token-2", token3)
		assert.Equal(t, 2, callCount)
	})

	t.Run("invalidateToken only affects specified scope", func(t *testing.T) {
		callCount := 0
		fakeClient := &dynamicFakeTokenServiceClient{
			getToken: func() string {
				callCount++
				return fmt.Sprintf("token-%d", callCount)
			},
		}

		provider := newTestTokenProvider(fakeClient, "test-cluster-id")

		// Cache tokens for two scopes
		_, err := provider.getTokenForScope(context.Background(), "scope-a")
		require.NoError(t, err)
		_, err = provider.getTokenForScope(context.Background(), "scope-b")
		require.NoError(t, err)
		assert.Equal(t, 2, callCount)

		// Invalidate only scope-a
		provider.invalidateToken("scope-a")

		// scope-a should get new token
		tokenA, err := provider.getTokenForScope(context.Background(), "scope-a")
		require.NoError(t, err)
		assert.Equal(t, "token-3", tokenA)
		assert.Equal(t, 3, callCount)

		// scope-b should still be cached
		tokenB, err := provider.getTokenForScope(context.Background(), "scope-b")
		require.NoError(t, err)
		assert.Equal(t, "token-2", tokenB)
		assert.Equal(t, 3, callCount)
	})
}

func TestScopedTokenTransport_RetryWithRequestBody(t *testing.T) {
	t.Run("POST request with body retries successfully", func(t *testing.T) {
		tokenCallCount := 0
		fakeClient := &dynamicFakeTokenServiceClient{
			getToken: func() string {
				tokenCallCount++
				return fmt.Sprintf("token-%d", tokenCallCount)
			},
		}

		bodiesReceived := []string{}
		requestCount := 0
		mockBase := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			requestCount++
			// Read the body to verify it's available
			body, _ := io.ReadAll(req.Body)
			bodiesReceived = append(bodiesReceived, string(body))

			if req.Header.Get("Authorization") == "Bearer token-1" {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error":"unauthorized"}`)),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		transport := &scopedTokenTransport{
			base:          mockBase,
			tokenProvider: newTestTokenProvider(fakeClient, "test-cluster-id"),
		}

		// Create request with body (body is buffered upfront, so retry works).
		bodyContent := `{"data":"test"}`
		req := httptest.NewRequest(http.MethodPost, "/v1/alerts", strings.NewReader(bodyContent))
		req.Header.Set(stackroxNamespaceHeader, "test-namespace")

		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "should return success after retry")
		assert.Equal(t, 2, tokenCallCount, "should have requested two tokens")
		assert.Equal(t, 2, requestCount, "should have made two requests")
		// Both requests should have received the body
		assert.Equal(t, []string{bodyContent, bodyContent}, bodiesReceived, "both requests should receive the body")
	})

	t.Run("first response body is drained and closed before retry", func(t *testing.T) {
		tokenCallCount := 0
		fakeClient := &dynamicFakeTokenServiceClient{
			getToken: func() string {
				tokenCallCount++
				return fmt.Sprintf("token-%d", tokenCallCount)
			},
		}

		firstBodyDrained := false
		firstBodyClosed := false
		mockBase := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("Authorization") == "Bearer token-1" {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body: &trackingReadCloser{
						ReadCloser: io.NopCloser(strings.NewReader(`{"error":"unauthorized"}`)),
						onRead:     func() { firstBodyDrained = true },
						onClose:    func() { firstBodyClosed = true },
					},
					Header: http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		transport := &scopedTokenTransport{
			base:          mockBase,
			tokenProvider: newTestTokenProvider(fakeClient, "test-cluster-id"),
		}

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		req.Header.Set(stackroxNamespaceHeader, "test-namespace")

		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.True(t, firstBodyDrained, "first response body should be drained before retry")
		assert.True(t, firstBodyClosed, "first response body should be closed before retry")
	})
}

// trackingReadCloser wraps an io.ReadCloser and tracks read/close operations.
type trackingReadCloser struct {
	io.ReadCloser
	onRead  func()
	onClose func()
}

func (t *trackingReadCloser) Read(p []byte) (n int, err error) {
	if t.onRead != nil {
		t.onRead()
	}
	return t.ReadCloser.Read(p)
}

func (t *trackingReadCloser) Close() error {
	if t.onClose != nil {
		t.onClose()
	}
	return t.ReadCloser.Close()
}

// roundTripperFunc is a helper to create RoundTripper from a function.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
