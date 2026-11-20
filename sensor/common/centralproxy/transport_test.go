package centralproxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	apiV1 "github.com/stackrox/rox/generated/api/v1"
	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sync"
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

	// callCount tracks the number of RPC calls made (optional, set to non-nil to enable)
	callCount *atomic.Int32
}

func (f *fakeTokenServiceClient) GenerateTokenForPermissionsAndScope(
	ctx context.Context,
	in *centralv1.GenerateTokenForPermissionsAndScopeRequest,
	opts ...grpc.CallOption,
) (*centralv1.GenerateTokenForPermissionsAndScopeResponse, error) {
	if f.callCount != nil {
		f.callCount.Add(1)
	}
	f.lastRequest = in
	return f.response, f.err
}

// newTestTokenProvider creates a tokenProvider for testing with the given client.
func newTestTokenProvider(client centralv1.TokenServiceClient, clusterID string) *tokenProvider {
	tp := newTokenProvider(&fakeClusterIDGetter{clusterID: clusterID})
	if client != nil {
		tp.client.Store(&client)
	}
	return tp
}

func consoleTestProfile(scope string) *proxyTokenProfile {
	return &proxyTokenProfile{
		identity: clientAuthKubernetesBearer + ":test-user", permissionSets: consolePermissionSets(),
		clusterScope: scope, legacyConsole: true,
	}
}

func metricsTestProfile() *proxyTokenProfile {
	return &proxyTokenProfile{
		identity:       clientAuthMTLS + ":" + env.SecureMetricsClientCertCN.Setting(),
		permissionSets: metricsPermissionSets(), clusterScope: FullClusterAccessScope,
	}
}

func TestScopedTokenTransport_RoundTrip(t *testing.T) {
	tests := []struct {
		name           string
		namespaceScope string
		expectedToken  string
	}{
		// #nosec G101 -- test data
		{
			name:           "empty scope",
			namespaceScope: "",
			expectedToken:  "token-for-empty-scope",
		},
		// #nosec G101 -- test data
		{
			name:           "specific namespace",
			namespaceScope: "my-namespace",
			expectedToken:  "token-for-namespace",
		},
		// #nosec G101 -- test data
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
			req = req.WithContext(withProxyTokenProfile(req.Context(), consoleTestProfile(tt.namespaceScope)))

			resp, err := transport.RoundTrip(req)
			require.NoError(t, err)
			require.NotNil(t, resp)

			assert.Equal(t, "Bearer "+tt.expectedToken, capturedAuthHeader)
		})
	}
}

func TestMetricsTokenDoesNotReuseConsoleScope(t *testing.T) {
	var calls atomic.Int32
	client := &fakeTokenServiceClient{
		response:  &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "metrics-token"},
		callCount: &calls,
	}
	provider := newTestTokenProvider(client, "sensor-cluster")
	_, err := provider.getTokenForProfile(context.Background(), consoleTestProfile(""))
	require.NoError(t, err)
	_, err = provider.getTokenForProfile(context.Background(), metricsTestProfile())
	require.NoError(t, err)
	assert.EqualValues(t, 2, calls.Load())
	require.Len(t, client.lastRequest.GetPermissionSets(), 1)
	assert.Equal(t, accesscontrol.OpenShiftPrometheus, client.lastRequest.GetPermissionSets()[0].GetName())
	protoassert.SlicesEqual(t, []*apiV1.HTTPRequestScope{{Path: "/metrics", Methods: []string{"GET"}}},
		client.lastRequest.GetPermissionSets()[0].GetHttpRequestScopes())
	assert.Empty(t, client.lastRequest.GetPermissions())
	protoassert.SlicesEqual(t, []*centralv1.ClusterScope{{ClusterId: "sensor-cluster", FullClusterAccess: true}},
		client.lastRequest.GetClusterScopes())

	transport := &scopedTokenTransport{
		base: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			assert.Empty(t, req.Header.Get(stackroxNamespaceHeader))
			assert.Equal(t, "Bearer metrics-token", req.Header.Get("Authorization"))
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		}),
		tokenProvider: provider,
	}
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set(stackroxNamespaceHeader, "other-cluster")
	req = req.WithContext(withProxyTokenProfile(req.Context(), metricsTestProfile()))
	_, err = transport.RoundTrip(req)
	require.NoError(t, err)
	assert.EqualValues(t, 2, calls.Load())
}

func TestProfileTokenIsBoundToConfiguredClaimsInCache(t *testing.T) {
	var calls atomic.Int32
	client := &fakeTokenServiceClient{
		response:  &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "profile-token"},
		callCount: &calls,
	}
	provider := newTokenProvider(&fakeClusterIDGetter{clusterID: "sensor-cluster"})
	var serviceClient centralv1.TokenServiceClient = client
	provider.client.Store(&serviceClient)
	profile := &proxyTokenProfile{
		identity: "system:serviceaccount:observability:reader",
		permissionSets: []*centralv1.ScopedPermissionSet{{
			Selector:          &centralv1.ScopedPermissionSet_Name{Name: "read-only-set"},
			HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/v1/metadata", Methods: []string{"GET"}}},
		}},
		clusterScope: FullClusterAccessScope,
	}
	token, err := provider.getTokenForProfile(context.Background(), profile)
	require.NoError(t, err)
	assert.Equal(t, "profile-token", token)
	assert.Equal(t, profile.identity, client.lastRequest.GetRequester())
	protoassert.SlicesEqual(t, profile.permissionSets, client.lastRequest.GetPermissionSets())
	require.Len(t, client.lastRequest.GetClusterScopes(), 1)
	assert.Equal(t, "sensor-cluster", client.lastRequest.GetClusterScopes()[0].GetClusterId())
	assert.True(t, client.lastRequest.GetClusterScopes()[0].GetFullClusterAccess())
	_, err = provider.getTokenForProfile(context.Background(), profile)
	require.NoError(t, err)
	assert.EqualValues(t, 1, calls.Load())

	other := *profile
	other.identity = "system:serviceaccount:observability:other"
	_, err = provider.getTokenForProfile(context.Background(), &other)
	require.NoError(t, err)
	assert.EqualValues(t, 2, calls.Load())
	other = *profile
	other.permissionSets = []*centralv1.ScopedPermissionSet{{
		Selector:          &centralv1.ScopedPermissionSet_Name{Name: "read-only-set"},
		HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/v1/alerts", Methods: []string{"GET"}}},
	}}
	_, err = provider.getTokenForProfile(context.Background(), &other)
	require.NoError(t, err)
	assert.EqualValues(t, 3, calls.Load())
	other = *profile
	other.permissionSets = []*centralv1.ScopedPermissionSet{{
		Selector: &centralv1.ScopedPermissionSet_Name{Name: "another-read-only-set"}, HttpRequestScopes: profile.permissionSets[0].GetHttpRequestScopes(),
	}}
	_, err = provider.getTokenForProfile(context.Background(), &other)
	require.NoError(t, err)
	assert.EqualValues(t, 4, calls.Load())
	protoassert.SlicesEqual(t, other.permissionSets, client.lastRequest.GetPermissionSets())

	other = *profile
	other.clusterScope = "team-a"
	_, err = provider.getTokenForProfile(context.Background(), &other)
	require.NoError(t, err)
	assert.EqualValues(t, 5, calls.Load())
	protoassert.SlicesEqual(t, []*centralv1.ClusterScope{{ClusterId: "sensor-cluster", Namespaces: []string{"team-a"}}},
		client.lastRequest.GetClusterScopes())
	other.clusterScope = "team-b"
	_, err = provider.getTokenForProfile(context.Background(), &other)
	require.NoError(t, err)
	assert.EqualValues(t, 6, calls.Load())
	protoassert.SlicesEqual(t, []*centralv1.ClusterScope{{ClusterId: "sensor-cluster", Namespaces: []string{"team-b"}}},
		client.lastRequest.GetClusterScopes())

	other = *profile
	other.permissionSets = []*centralv1.ScopedPermissionSet{{
		Selector:          &centralv1.ScopedPermissionSet_Id{Id: accesscontrol.DefaultPermissionSetIDs[accesscontrol.K8sCentralProxyUser]},
		HttpRequestScopes: profile.permissionSets[0].GetHttpRequestScopes(),
	}}
	_, err = provider.getTokenForProfile(context.Background(), &other)
	require.NoError(t, err)
	assert.EqualValues(t, 7, calls.Load())
	protoassert.SlicesEqual(t, other.permissionSets, client.lastRequest.GetPermissionSets())

	other = *profile
	other.legacyConsole = true
	_, err = provider.getTokenForProfile(context.Background(), &other)
	require.NoError(t, err)
	assert.EqualValues(t, 8, calls.Load(), "legacy fallback claims must have a separate cache entry")

	provider.clusterIDGetter = &fakeClusterIDGetter{clusterID: "another-cluster"}
	_, err = provider.getTokenForProfile(context.Background(), profile)
	require.NoError(t, err)
	assert.EqualValues(t, 9, calls.Load(), "cluster identity must be part of the cache key")
	assert.Equal(t, "another-cluster", client.lastRequest.GetClusterScopes()[0].GetClusterId())
}

func TestPerAccessProfileSharesTokenAcrossRoutesButNotNamespaces(t *testing.T) {
	var calls atomic.Int32
	client := &fakeTokenServiceClient{
		response:  &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "per-access-token"},
		callCount: &calls,
	}
	provider := newTokenProvider(&fakeClusterIDGetter{clusterID: "sensor-cluster"})
	var serviceClient centralv1.TokenServiceClient = client
	provider.client.Store(&serviceClient)
	profile := &proxyTokenProfile{
		identity: "kubernetesBearer:alice",
		permissionSets: []*centralv1.ScopedPermissionSet{
			{Selector: &centralv1.ScopedPermissionSet_Name{Name: "Deployment Reader"}, HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/v1/deployments", Methods: []string{"GET"}}}},
			{Selector: &centralv1.ScopedPermissionSet_Name{Name: "Image Reader"}, HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/v1/images", Methods: []string{"GET"}}}},
		},
		clusterScope: "team-a",
	}
	_, err := provider.getTokenForProfile(t.Context(), profile)
	require.NoError(t, err)
	assert.Empty(t, client.lastRequest.GetPermissions())
	protoassert.SlicesEqual(t, profile.permissionSets, client.lastRequest.GetPermissionSets())
	other := *profile
	_, err = provider.getTokenForProfile(t.Context(), &other)
	require.NoError(t, err)
	assert.EqualValues(t, 1, calls.Load())
	other.clusterScope = "team-b"
	_, err = provider.getTokenForProfile(t.Context(), &other)
	require.NoError(t, err)
	assert.EqualValues(t, 2, calls.Load())

	other = *profile
	other.permissionSets = []*centralv1.ScopedPermissionSet{
		{Selector: &centralv1.ScopedPermissionSet_Id{Id: accesscontrol.DefaultPermissionSetIDs[accesscontrol.K8sCentralProxyUser]}, HttpRequestScopes: profile.permissionSets[0].GetHttpRequestScopes()},
		profile.permissionSets[1],
	}
	_, err = provider.getTokenForProfile(t.Context(), &other)
	require.NoError(t, err)
	assert.EqualValues(t, 3, calls.Load())
	protoassert.SlicesEqual(t, other.permissionSets, client.lastRequest.GetPermissionSets())
}

func TestProfileTransportReplacesCallerCredentials(t *testing.T) {
	client := &fakeTokenServiceClient{
		response: &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "issued-token"},
	}
	provider := newTokenProvider(&fakeClusterIDGetter{clusterID: "sensor-cluster"})
	var serviceClient centralv1.TokenServiceClient = client
	provider.client.Store(&serviceClient)
	transport := &scopedTokenTransport{
		base: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "Bearer issued-token", req.Header.Get("Authorization"))
			assert.Empty(t, req.Header.Get(stackroxNamespaceHeader))
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		}),
		tokenProvider: provider,
	}
	profile := &proxyTokenProfile{
		identity: "user:test-user",
		permissionSets: []*centralv1.ScopedPermissionSet{{
			Selector:          &centralv1.ScopedPermissionSet_Name{Name: "read-only-set"},
			HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/v1/metadata", Methods: []string{"GET"}}},
		}},
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/metadata", nil)
	req.Header.Set("Authorization", "Bearer caller-kubernetes-token")
	req.Header.Set(stackroxNamespaceHeader, "test-namespace")
	req = req.WithContext(withProxyTokenProfile(req.Context(), profile))
	response, err := transport.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "Bearer caller-kubernetes-token", req.Header.Get("Authorization"))
}

func TestProfileTokenUnifiedGrantsAndLegacyFallback(t *testing.T) {
	console := &centralv1.ScopedPermissionSet{
		Selector: &centralv1.ScopedPermissionSet_Id{Id: accesscontrol.DefaultPermissionSetIDs[accesscontrol.K8sCentralProxyUser]}, HttpRequestScopes: consoleRequestScopes(),
	}
	unrestricted := &centralv1.ScopedPermissionSet{Selector: &centralv1.ScopedPermissionSet_Name{Name: "Reader"}}
	for name, tc := range map[string]struct {
		grants   []*centralv1.ScopedPermissionSet
		fallback bool
	}{
		"unrestricted":        {grants: []*centralv1.ScopedPermissionSet{unrestricted}},
		"mixed console first": {grants: []*centralv1.ScopedPermissionSet{console, unrestricted}},
		"mixed custom first":  {grants: []*centralv1.ScopedPermissionSet{unrestricted, console}},
		"new console":         {grants: []*centralv1.ScopedPermissionSet{console}},
		"legacy console":      {grants: []*centralv1.ScopedPermissionSet{console}, fallback: true},
	} {
		t.Run(name, func(t *testing.T) {
			client := &fakeTokenServiceClient{
				response: &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "token"},
			}
			provider := newTokenProvider(&fakeClusterIDGetter{clusterID: "sensor-cluster"})
			profile := &proxyTokenProfile{
				identity: "kubernetesBearer:alice", permissionSets: tc.grants,
				clusterScope: "team-a", legacyConsole: tc.fallback,
			}
			_, err := provider.requestProfileToken(t.Context(), client, profile)
			require.NoError(t, err)
			protoassert.SlicesEqual(t, []*centralv1.ClusterScope{{ClusterId: "sensor-cluster", Namespaces: []string{"team-a"}}},
				client.lastRequest.GetClusterScopes())
			if tc.fallback {
				assert.Empty(t, client.lastRequest.GetPermissionSets())
				assert.Equal(t, map[string]centralv1.Access{
					"Deployment": centralv1.Access_READ_ACCESS,
					"Image":      centralv1.Access_READ_ACCESS,
				}, client.lastRequest.GetPermissions())
			} else {
				protoassert.SlicesEqual(t, tc.grants, client.lastRequest.GetPermissionSets())
				assert.Empty(t, client.lastRequest.GetPermissions())
			}
		})
	}
}

func TestScopedTokenTransport_RoundTrip_Error(t *testing.T) {
	t.Run("missing authorized profile fails closed", func(t *testing.T) {
		transport := &scopedTokenTransport{}
		response, err := transport.RoundTrip(httptest.NewRequest(http.MethodGet, "/metrics", nil))
		require.ErrorContains(t, err, "profile is missing")
		assert.Nil(t, response)
	})

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
		req = req.WithContext(withProxyTokenProfile(req.Context(), consoleTestProfile("")))
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
		req = req.WithContext(withProxyTokenProfile(req.Context(), consoleTestProfile("")))
		resp, err := transport.RoundTrip(req)

		assert.ErrorIs(t, err, baseErr)
		assert.Nil(t, resp)
	})
}

func TestTokenProvider_ConsoleProfileScopes(t *testing.T) {
	tests := []struct {
		name            string
		namespaceScope  string
		clusterID       string
		wantScopes      bool
		wantFullAccess  bool
		wantNamespaces  []string
		perAccessTokens bool
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
			name:            "Central with per-access tokens",
			namespaceScope:  "my-namespace",
			clusterID:       "test-cluster-id",
			wantScopes:      true,
			wantNamespaces:  []string{"my-namespace"},
			perAccessTokens: true,
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
			profile := consoleTestProfile(tt.namespaceScope)
			profile.legacyConsole = !tt.perAccessTokens
			token, err := provider.getTokenForProfile(context.Background(), profile)
			require.NoError(t, err)
			assert.Equal(t, "test-token-123", token)

			// Verify request
			req := fakeClient.lastRequest
			require.NotNil(t, req)
			assert.Equal(t, tokenTTL, req.GetLifetime().AsDuration())

			if tt.perAccessTokens {
				require.Len(t, req.GetPermissionSets(), 1)
				assert.Equal(t, accesscontrol.DefaultPermissionSetIDs[accesscontrol.K8sCentralProxyUser], req.GetPermissionSets()[0].GetId())
				assert.Empty(t, req.GetPermissions())
			} else {
				assert.Empty(t, req.GetPermissionSets())
				assert.Equal(t, map[string]centralv1.Access{
					"Deployment": centralv1.Access_READ_ACCESS,
					"Image":      centralv1.Access_READ_ACCESS,
				}, req.GetPermissions())
			}

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

type contextTokenServiceClient struct {
	request func(context.Context) (string, error)
}

func (c *contextTokenServiceClient) GenerateTokenForPermissionsAndScope(
	ctx context.Context, _ *centralv1.GenerateTokenForPermissionsAndScopeRequest, _ ...grpc.CallOption,
) (*centralv1.GenerateTokenForPermissionsAndScopeResponse, error) {
	token, err := c.request(ctx)
	return &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: token}, err
}

func TestTokenProviderSharedRequestHasIndependentDeadline(t *testing.T) {
	started := make(chan context.Context, 1)
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	var calls atomic.Int32
	client := &contextTokenServiceClient{request: func(ctx context.Context) (string, error) {
		calls.Add(1)
		started <- ctx
		<-release
		return "shared-token", nil
	}}
	provider := newTestTokenProvider(client, "sensor-cluster")
	profile := consoleTestProfile("team-a")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	finished := make(chan error, 1)
	go func() {
		_, err := provider.getTokenForProfile(ctx, profile)
		finished <- err
	}()
	requestCtx := <-started
	deadline, ok := requestCtx.Deadline()
	require.True(t, ok, "shared RPC must have a bounded lifetime")
	assert.WithinDuration(t, time.Now().Add(tokenRequestTimeout), deadline, time.Second)
	cancel()
	require.ErrorIs(t, <-finished, context.Canceled)
	assert.NoError(t, requestCtx.Err(), "caller cancellation must not cancel the shared RPC")
	close(release)
	token, err := provider.getTokenForProfile(t.Context(), profile)
	require.NoError(t, err)
	assert.Equal(t, "shared-token", token)
	assert.EqualValues(t, 1, calls.Load())
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
		token1, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("namespace-a"))
		require.NoError(t, err)
		assert.Equal(t, "cached-token", token1)
		assert.Equal(t, 1, callCount)

		// Second call with same scope should use cache
		token2, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("namespace-a"))
		require.NoError(t, err)
		assert.Equal(t, "cached-token", token2)
		assert.Equal(t, 1, callCount, "should use cached token")

		// Third call with different scope should hit API again
		token3, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("namespace-b"))
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
		tokenA, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("scope-a"))
		require.NoError(t, err)
		assert.Equal(t, "token-1", tokenA)

		// Request for scope B (different scope, should get new token)
		tokenB, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("scope-b"))
		require.NoError(t, err)
		assert.Equal(t, "token-2", tokenB)

		// Request for scope A again (should use cached)
		tokenACached, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("scope-a"))
		require.NoError(t, err)
		assert.Equal(t, "token-1", tokenACached)

		// Request for scope B again (should use cached)
		tokenBCached, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("scope-b"))
		require.NoError(t, err)
		assert.Equal(t, "token-2", tokenBCached)
	})
}

func TestTokenProvider_ErrorHandling(t *testing.T) {
	t.Run("no client returns error", func(t *testing.T) {
		provider := newTestTokenProvider(nil, "test-cluster-id")

		_, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("namespace"))
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

		_, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("namespace"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty proxy client token")

		// Ensure nothing was cached for this scope
		_, found := provider.profileTokenCache.Get("test-cluster-id:" + profileCacheKey(consoleTestProfile("namespace")))
		assert.False(t, found)
	})

	t.Run("client error is returned and not cached", func(t *testing.T) {
		fakeErr := errors.New("rpc failure")

		fakeClient := &fakeTokenServiceClient{
			err: fakeErr,
		}

		provider := newTestTokenProvider(fakeClient, "test-cluster-id")

		_, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("namespace-error"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rpc failure")

		// Ensure token is not cached for the failing scope
		_, found := provider.profileTokenCache.Get("test-cluster-id:" + profileCacheKey(consoleTestProfile("namespace-error")))
		assert.False(t, found)
	})
}

func TestTokenProviderRequiresClusterID(t *testing.T) {
	client := &fakeTokenServiceClient{}
	provider := newTestTokenProvider(client, "")
	token, err := provider.getTokenForProfile(t.Context(), consoleTestProfile("namespace"))
	require.Error(t, err)
	assert.Empty(t, token)
	assert.Nil(t, client.lastRequest)
	assert.Contains(t, err.Error(), "cluster ID not available")
}

func TestConsoleRequestScopes(t *testing.T) {
	scopes := consoleRequestScopes()
	require.NotEmpty(t, scopes)
	actual, err := tokens.HTTPRequestScopesFromProto(scopes)
	require.NoError(t, err)
	assert.Equal(t, centralproxy.ConsoleHTTPScopes, actual)

	for i := 1; i < len(scopes); i++ {
		assert.Less(t, scopes[i-1].GetPath(), scopes[i].GetPath())
	}

	scopes[0].Methods[0] = "PUT"
	assert.NotEqual(t, "PUT", centralproxy.ConsoleHTTPScopes[scopes[0].GetPath()][0])
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
		req = req.WithContext(withProxyTokenProfile(req.Context(), consoleTestProfile("test-namespace")))

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
		req = req.WithContext(withProxyTokenProfile(req.Context(), consoleTestProfile("test-namespace")))

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
		req = req.WithContext(withProxyTokenProfile(req.Context(), consoleTestProfile("test-namespace")))

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
				req = req.WithContext(withProxyTokenProfile(req.Context(), consoleTestProfile("test-namespace")))

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
		req = req.WithContext(withProxyTokenProfile(req.Context(), consoleTestProfile("test-namespace")))

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

func TestTokenProvider_InvalidateProfile(t *testing.T) {
	t.Run("invalidation removes token from cache", func(t *testing.T) {
		callCount := 0
		fakeClient := &dynamicFakeTokenServiceClient{
			getToken: func() string {
				callCount++
				return fmt.Sprintf("token-%d", callCount)
			},
		}

		provider := newTestTokenProvider(fakeClient, "test-cluster-id")

		// Get token (causes cache)
		token1, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("my-scope"))
		require.NoError(t, err)
		assert.Equal(t, "token-1", token1)
		assert.Equal(t, 1, callCount)

		// Get again - should be cached
		token2, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("my-scope"))
		require.NoError(t, err)
		assert.Equal(t, "token-1", token2)
		assert.Equal(t, 1, callCount)

		// Invalidate
		provider.invalidateProfile(consoleTestProfile("my-scope"))

		// Get again - should fetch new token
		token3, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("my-scope"))
		require.NoError(t, err)
		assert.Equal(t, "token-2", token3)
		assert.Equal(t, 2, callCount)
	})

	t.Run("invalidation only affects specified scope", func(t *testing.T) {
		callCount := 0
		fakeClient := &dynamicFakeTokenServiceClient{
			getToken: func() string {
				callCount++
				return fmt.Sprintf("token-%d", callCount)
			},
		}

		provider := newTestTokenProvider(fakeClient, "test-cluster-id")

		// Cache tokens for two scopes
		_, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("scope-a"))
		require.NoError(t, err)
		_, err = provider.getTokenForProfile(context.Background(), consoleTestProfile("scope-b"))
		require.NoError(t, err)
		assert.Equal(t, 2, callCount)

		// Invalidate only scope-a
		provider.invalidateProfile(consoleTestProfile("scope-a"))

		// scope-a should get new token
		tokenA, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("scope-a"))
		require.NoError(t, err)
		assert.Equal(t, "token-3", tokenA)
		assert.Equal(t, 3, callCount)

		// scope-b should still be cached
		tokenB, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("scope-b"))
		require.NoError(t, err)
		assert.Equal(t, "token-2", tokenB)
		assert.Equal(t, 3, callCount)
	})

	t.Run("metrics invalidation leaves an equal-named scope cached", func(t *testing.T) {
		var calls atomic.Int32
		client := &dynamicFakeTokenServiceClient{
			getToken: func() string { return fmt.Sprintf("token-%d", calls.Add(1)) },
		}
		provider := newTestTokenProvider(client, "test-cluster-id")
		key := accesscontrol.OpenShiftPrometheus

		scopeToken, err := provider.getTokenForProfile(context.Background(), consoleTestProfile(key))
		require.NoError(t, err)
		metricsToken, err := provider.getTokenForProfile(context.Background(), metricsTestProfile())
		require.NoError(t, err)
		require.NotEqual(t, scopeToken, metricsToken)

		provider.invalidateProfile(metricsTestProfile())
		cachedScopeToken, err := provider.getTokenForProfile(context.Background(), consoleTestProfile(key))
		require.NoError(t, err)
		assert.Equal(t, scopeToken, cachedScopeToken)
		freshMetricsToken, err := provider.getTokenForProfile(context.Background(), metricsTestProfile())
		require.NoError(t, err)
		assert.NotEqual(t, metricsToken, freshMetricsToken)
		assert.EqualValues(t, 3, calls.Load())
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
		req = req.WithContext(withProxyTokenProfile(req.Context(), consoleTestProfile("test-namespace")))

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
		req = req.WithContext(withProxyTokenProfile(req.Context(), consoleTestProfile("test-namespace")))

		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.True(t, firstBodyDrained, "first response body should be drained before retry")
		assert.True(t, firstBodyClosed, "first response body should be closed before retry")
	})
}

func TestScopedTokenTransport_RequestBodySizeLimit(t *testing.T) {
	tests := map[string]struct {
		bodySize  int
		wantError bool
	}{
		"body within limit is accepted":     {bodySize: 64, wantError: false},
		"body exactly at limit is accepted": {bodySize: maxRequestBodySize, wantError: false},
		"body exceeding limit is rejected":  {bodySize: maxRequestBodySize + 1, wantError: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fakeClient := &fakeTokenServiceClient{
				response: &centralv1.GenerateTokenForPermissionsAndScopeResponse{
					Token: "test-token",
				},
			}

			mockBase := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if tt.wantError {
					t.Fatal("base transport should not be called for oversized body")
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

			body := strings.NewReader(strings.Repeat("x", tt.bodySize))
			req := httptest.NewRequest(http.MethodPost, "/v1/alerts", body)
			req = req.WithContext(withProxyTokenProfile(req.Context(), consoleTestProfile("")))
			resp, err := transport.RoundTrip(req)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), "maximum allowed size")
			} else {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		})
	}
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

// barrierFakeTokenServiceClient allows controlling when the token is returned using a barrier.
// This is used for deterministic testing of request coalescing without timing dependencies.
type barrierFakeTokenServiceClient struct {
	getToken func() string
	barrier  <-chan struct{} // If non-nil, blocks until closed before returning
}

func (b *barrierFakeTokenServiceClient) GenerateTokenForPermissionsAndScope(
	ctx context.Context,
	in *centralv1.GenerateTokenForPermissionsAndScopeRequest,
	opts ...grpc.CallOption,
) (*centralv1.GenerateTokenForPermissionsAndScopeResponse, error) {
	if b.barrier != nil {
		<-b.barrier
	}
	return &centralv1.GenerateTokenForPermissionsAndScopeResponse{
		Token: b.getToken(),
	}, nil
}

func TestTokenProvider_Coalescing(t *testing.T) {
	t.Run("equal scope and permission-set keys do not coalesce", func(t *testing.T) {
		client := &fakeTokenServiceClient{
			response: &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "metrics-token"},
		}
		provider := newTestTokenProvider(client, "test-cluster-id")
		key := accesscontrol.OpenShiftPrometheus
		started := make(chan struct{})
		release := make(chan struct{})
		finished := make(chan struct{})
		defer func() {
			close(release)
			<-finished
		}()
		go func() {
			defer close(finished)
			_, _ = provider.profileTokenGroup.Coalesce(context.Background(),
				"test-cluster-id:"+profileCacheKey(consoleTestProfile(key)), func() (string, error) {
					close(started)
					<-release
					return "console-token", nil
				})
		}()
		<-started

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		token, err := provider.getTokenForProfile(ctx, metricsTestProfile())
		require.NoError(t, err)
		assert.Equal(t, "metrics-token", token)
		require.Len(t, client.lastRequest.GetPermissionSets(), 1)
		assert.Equal(t, key, client.lastRequest.GetPermissionSets()[0].GetName())
	})

	t.Run("concurrent requests for same scope make only one RPC call", func(t *testing.T) {
		var callCount atomic.Int32
		var wg sync.WaitGroup

		// Barrier to keep the RPC in-flight while goroutines start.
		// This ensures deterministic coalescing behavior without relying on timing.
		barrier := make(chan struct{})

		fakeClient := &barrierFakeTokenServiceClient{
			barrier: barrier,
			getToken: func() string {
				callCount.Add(1)
				return "coalesced-token"
			},
		}

		provider := newTestTokenProvider(fakeClient, "test-cluster-id")

		const numGoroutines = 10
		tokens := make([]string, numGoroutines)
		errs := make([]error, numGoroutines)

		// Use a separate WaitGroup to track when all goroutines have started
		var startWg sync.WaitGroup
		startWg.Add(numGoroutines)

		// Launch concurrent requests for the same scope
		for i := range numGoroutines {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				startWg.Done() // Signal this goroutine has started
				tokens[idx], errs[idx] = provider.getTokenForProfile(context.Background(), consoleTestProfile("shared-scope"))
			}(i)
		}

		// Wait for all goroutines to start, then release the barrier
		startWg.Wait()
		close(barrier)
		wg.Wait()

		// Verify only ONE RPC call was made
		assert.Equal(t, int32(1), callCount.Load(),
			"expected exactly 1 RPC call for %d concurrent requests", numGoroutines)

		// Verify all goroutines got the same token
		for i := range numGoroutines {
			require.NoError(t, errs[i])
			assert.Equal(t, "coalesced-token", tokens[i])
		}
	})

	t.Run("errors are not shared - each caller can retry", func(t *testing.T) {
		var callCount atomic.Int32
		fakeClient := &fakeTokenServiceClient{
			callCount: &callCount,
		}

		provider := newTestTokenProvider(fakeClient, "test-cluster-id")

		// First request fails
		fakeClient.err = errors.New("transient failure")
		_, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("error-scope"))
		require.Error(t, err)

		// singleflight removes the key after the call completes with error
		// so a second request will trigger a new RPC call
		fakeClient.err = nil
		fakeClient.response = &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "recovered-token"}

		token, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("error-scope"))
		require.NoError(t, err)
		assert.Equal(t, "recovered-token", token)
		assert.Equal(t, int32(2), callCount.Load(), "expected two RPC calls (one failure, one success)")
	})

	t.Run("invalidation allows fresh RPC after invalidation", func(t *testing.T) {
		var callCount atomic.Int32

		fakeClient := &dynamicFakeTokenServiceClient{
			getToken: func() string {
				count := callCount.Add(1)
				return fmt.Sprintf("token-%d", count)
			},
		}

		provider := newTestTokenProvider(fakeClient, "test-cluster-id")

		// First request
		token1, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("invalidate-scope"))
		require.NoError(t, err)
		assert.Equal(t, "token-1", token1)
		assert.Equal(t, int32(1), callCount.Load())

		// Invalidate
		provider.invalidateProfile(consoleTestProfile("invalidate-scope"))

		// Second request should trigger new RPC
		token2, err := provider.getTokenForProfile(context.Background(), consoleTestProfile("invalidate-scope"))
		require.NoError(t, err)
		assert.Equal(t, "token-2", token2)
		assert.Equal(t, int32(2), callCount.Load())
	})
}
