package centralproxy

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"
)

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		centralReachable bool
		wantStatusCode   int
		wantError        string
	}{
		{
			name:             "GET is allowed",
			method:           http.MethodGet,
			centralReachable: true,
			wantStatusCode:   0, // validateRequest returns nil on success
		},
		{
			name:             "POST is allowed",
			method:           http.MethodPost,
			centralReachable: true,
			wantStatusCode:   0,
		},
		{
			name:             "OPTIONS is allowed (for CORS preflight)",
			method:           http.MethodOptions,
			centralReachable: true,
			wantStatusCode:   0,
		},
		{
			name:             "HEAD is allowed",
			method:           http.MethodHead,
			centralReachable: true,
			wantStatusCode:   0,
		},
		{
			name:             "PUT returns 405",
			method:           http.MethodPut,
			centralReachable: true,
			wantStatusCode:   http.StatusMethodNotAllowed,
			wantError:        "method PUT not allowed",
		},
		{
			name:             "DELETE returns 405",
			method:           http.MethodDelete,
			centralReachable: true,
			wantStatusCode:   http.StatusMethodNotAllowed,
			wantError:        "method DELETE not allowed",
		},
		{
			name:             "central not reachable returns 503",
			method:           http.MethodGet,
			centralReachable: false,
			wantStatusCode:   http.StatusServiceUnavailable,
			wantError:        "central not reachable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				clusterIDGetter: &testClusterIDGetter{clusterID: "test-cluster-id"},
			}
			h.centralReachable.Store(tt.centralReachable)

			req := httptest.NewRequest(tt.method, "/v1/alerts", nil)
			err := h.validateRequest(req)

			if tt.wantStatusCode == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				httpErr, ok := err.(pkghttputil.HTTPError)
				require.True(t, ok, "error should be an HTTPError")
				assert.Equal(t, tt.wantStatusCode, httpErr.HTTPStatusCode())
				assert.Contains(t, err.Error(), tt.wantError)
			}
		})
	}
}

func TestServeHTTP(t *testing.T) {
	t.Run("validation fails, proxy not called", func(t *testing.T) {
		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		h := newTestHandler(t, baseURL, nil, nil, "test-token")
		h.centralReachable.Store(false) // Will fail validation

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "central not reachable")
	})

	t.Run("validation passes, request proxied", func(t *testing.T) {
		var proxyCalled bool
		mockTransport := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			proxyCalled = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		h := newTestHandler(t, baseURL, mockTransport, newAllowingAuthorizer(t), "test-token")
		h.centralReachable.Store(true)

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.True(t, proxyCalled, "proxy should be called")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("proxy error handled by ErrorHandler", func(t *testing.T) {
		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		h := newTestHandlerWithTransportError(t, baseURL, newAllowingAuthorizer(t), errTransportError)
		h.centralReachable.Store(true)

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "failed to contact central")
	})
}

func TestServeHTTP_ConstructsAbsoluteURLs(t *testing.T) {
	tests := []struct {
		name         string
		baseURL      string
		requestPath  string
		requestQuery string
		expectedURL  string
	}{
		{
			name:        "simple path without query",
			baseURL:     "https://central.stackrox.svc:443",
			requestPath: "/v1/alerts",
			expectedURL: "https://central.stackrox.svc:443/v1/alerts",
		},
		{
			name:         "path with query parameters",
			baseURL:      "https://central.stackrox.svc:443",
			requestPath:  "/v1/alerts",
			requestQuery: "limit=10&offset=20",
			expectedURL:  "https://central.stackrox.svc:443/v1/alerts?limit=10&offset=20",
		},
		{
			name:        "graphql endpoint",
			baseURL:     "https://central.stackrox.svc:443",
			requestPath: "/api/graphql",
			expectedURL: "https://central.stackrox.svc:443/api/graphql",
		},
		{
			name:        "endpoint without port",
			baseURL:     "https://central.stackrox.svc",
			requestPath: "/v1/deployments",
			expectedURL: "https://central.stackrox.svc/v1/deployments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, err := url.Parse(tt.baseURL)
			assert.NoError(t, err)

			var capturedURL string
			mockTransport := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				// Capture the URL that was used for the request
				capturedURL = req.URL.String()

				// Verify it's an absolute URL with scheme and host
				assert.NotEmpty(t, req.URL.Scheme, "URL scheme should not be empty")
				assert.NotEmpty(t, req.URL.Host, "URL host should not be empty")

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("{}")),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			})

			h := newTestHandler(t, baseURL, mockTransport, newAllowingAuthorizer(t), "test-token")
			h.centralReachable.Store(true)

			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			req.Header.Set("Authorization", "Bearer test-token")
			if tt.requestQuery != "" {
				req.URL.RawQuery = tt.requestQuery
			}
			writer := httptest.NewRecorder()
			h.ServeHTTP(writer, req)

			assert.Equal(t, tt.expectedURL, capturedURL)
		})
	}
}

func TestNotify(t *testing.T) {
	h := &Handler{}

	t.Run("CentralReachable sets centralReachable to true", func(t *testing.T) {
		h.centralReachable.Store(false)
		h.Notify(common.SensorComponentEventCentralReachable)
		assert.True(t, h.centralReachable.Load())
	})

	t.Run("OfflineMode sets centralReachable to false", func(t *testing.T) {
		h.centralReachable.Store(true)
		h.Notify(common.SensorComponentEventOfflineMode)
		assert.False(t, h.centralReachable.Load())
	})
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		wantToken   string
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid bearer token",
			authHeader: "Bearer my-secret-token-123",
			wantToken:  "my-secret-token-123",
			wantErr:    false,
		},
		{
			name:       "bearer token with spaces",
			authHeader: "Bearer token-with-spaces   ",
			wantToken:  "token-with-spaces   ",
			wantErr:    false,
		},
		{
			name:        "missing authorization header",
			authHeader:  "",
			wantErr:     true,
			errContains: "missing or invalid bearer token",
		},
		{
			name:        "invalid format - no Bearer prefix",
			authHeader:  "Basic dXNlcjpwYXNz",
			wantErr:     true,
			errContains: "missing or invalid bearer token",
		},
		{
			name:       "case-insensitive bearer prefix (lowercase)",
			authHeader: "bearer my-token-123",
			wantToken:  "my-token-123",
			wantErr:    false,
		},
		{
			name:       "case-insensitive bearer prefix (mixed case)",
			authHeader: "BeArEr my-token-123",
			wantToken:  "my-token-123",
			wantErr:    false,
		},
		{
			name:        "empty token after Bearer",
			authHeader:  "Bearer ",
			wantErr:     true,
			errContains: "missing or invalid bearer token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			token, err := extractBearerToken(req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantToken, token)
			}
		})
	}
}

func TestServeHTTP_AuthorizationIntegration(t *testing.T) {
	t.Run("authorization failure prevents proxy call", func(t *testing.T) {
		var proxyCalled bool
		mockTransport := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			proxyCalled = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		// Create a k8sAuthorizer with a fake client that denies access
		denyingAuthorizer := newDenyingAuthorizer(t)

		h := newTestHandler(t, baseURL, mockTransport, denyingAuthorizer, "test-token")
		h.centralReachable.Store(true)

		// Set namespace scope header to trigger SAR check
		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set(stackroxNamespaceHeader, "my-namespace")
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.False(t, proxyCalled, "proxy should not be called when authorization fails")
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "lacks")
	})

	t.Run("no authorizer returns server error", func(t *testing.T) {
		var proxyCalled bool
		mockTransport := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			proxyCalled = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		h := newTestHandler(t, baseURL, mockTransport, nil, "test-token")
		h.centralReachable.Store(true)

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.False(t, proxyCalled, "proxy should not be called when authorizer is not configured")
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "authorizer not configured")
	})

	t.Run("authorization success allows proxy call", func(t *testing.T) {
		var proxyCalled bool
		mockTransport := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			proxyCalled = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		h := newTestHandler(t, baseURL, mockTransport, newAllowingAuthorizer(t), "test-token")
		h.centralReachable.Store(true)

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.True(t, proxyCalled, "proxy should be called when authorization succeeds")
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestServeHTTP_NamespaceScopeBasedAuthorization(t *testing.T) {
	t.Run("empty namespace scope skips SAR check", func(t *testing.T) {
		var proxyCalled bool
		mockTransport := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			proxyCalled = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		// Create authorizer that denies SAR but allows TokenReview
		denyingAuthorizer := newDenyingAuthorizer(t)

		h := newTestHandler(t, baseURL, mockTransport, denyingAuthorizer, "test-token")
		h.centralReachable.Store(true)

		// No namespace scope header - should skip SAR
		req := httptest.NewRequest(http.MethodGet, "/v1/metadata", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		// NOT setting ACS-AUTH-NAMESPACE-SCOPE header
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.True(t, proxyCalled, "proxy should be called when namespace scope is empty (no SAR)")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("specific namespace scope triggers SAR check", func(t *testing.T) {
		var proxyCalled bool
		mockTransport := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			proxyCalled = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		denyingAuthorizer := newDenyingAuthorizer(t)

		h := newTestHandler(t, baseURL, mockTransport, denyingAuthorizer, "test-token")
		h.centralReachable.Store(true)

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set(stackroxNamespaceHeader, "my-namespace")
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.False(t, proxyCalled, "proxy should not be called when SAR fails for namespace scope")
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("cluster-wide scope (*) triggers SAR check", func(t *testing.T) {
		var proxyCalled bool
		mockTransport := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			proxyCalled = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		denyingAuthorizer := newDenyingAuthorizer(t)

		h := newTestHandler(t, baseURL, mockTransport, denyingAuthorizer, "test-token")
		h.centralReachable.Store(true)

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set(stackroxNamespaceHeader, FullClusterAccessScope)
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.False(t, proxyCalled, "proxy should not be called when SAR fails for cluster-wide scope")
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("namespace scope with valid permissions succeeds", func(t *testing.T) {
		var proxyCalled bool
		mockTransport := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			proxyCalled = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		fakeClient := fake.NewClientset()

		// Mock TokenReview to return authenticated
		fakeClient.PrependReactor("create", "tokenreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
			return true, &authenticationv1.TokenReview{
				Status: authenticationv1.TokenReviewStatus{
					Authenticated: true,
					User: authenticationv1.UserInfo{
						Username: "test-user",
					},
				},
			}, nil
		})

		// Mock SAR to allow
		fakeClient.PrependReactor("create", "subjectaccessreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
			return true, &authv1.SubjectAccessReview{
				Status: authv1.SubjectAccessReviewStatus{
					Allowed: true,
				},
			}, nil
		})

		h := newTestHandler(t, baseURL, mockTransport, newK8sAuthorizer(fakeClient), "test-token")
		h.centralReachable.Store(true)

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set(stackroxNamespaceHeader, "my-namespace")
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.True(t, proxyCalled, "proxy should be called with valid permissions")
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestServeHTTP_TransportFailure(t *testing.T) {
	t.Run("transport failure returns 500", func(t *testing.T) {
		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		h := newTestHandlerWithTransportError(t, baseURL, newAllowingAuthorizer(t), errTransportError)
		h.centralReachable.Store(true)

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "failed to contact central")
	})

	t.Run("initialization error returns 503", func(t *testing.T) {
		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		// Use errServiceUnavailable to simulate initialization failure
		h := newTestHandlerWithTransportError(t, baseURL, newAllowingAuthorizer(t), errServiceUnavailable)
		h.centralReachable.Store(true)

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "proxy temporarily unavailable")
	})
}

func TestServeHTTP_TokenInjection(t *testing.T) {
	t.Run("token is injected into proxied request", func(t *testing.T) {
		expectedToken := "dynamic-central-token-123"
		var capturedAuthHeader string

		mockTransport := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			capturedAuthHeader = req.Header.Get("Authorization")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		h := newTestHandler(t, baseURL, mockTransport, newAllowingAuthorizer(t), expectedToken)
		h.centralReachable.Store(true)

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		req.Header.Set("Authorization", "Bearer user-token") // User's incoming token
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "Bearer "+expectedToken, capturedAuthHeader, "proxied request should have Central token, not user token")
	})
}

func TestServeHTTP_RequiresAuthentication(t *testing.T) {
	t.Run("missing token returns 401", func(t *testing.T) {
		var proxyCalled bool
		mockTransport := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			proxyCalled = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		h := newTestHandler(t, baseURL, mockTransport, newAllowingAuthorizer(t), "test-token")
		h.centralReachable.Store(true)

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		// No Authorization header set
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.False(t, proxyCalled, "proxy should not be called without authorization header")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "bearer token")
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		var proxyCalled bool
		mockTransport := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			proxyCalled = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		baseURL, err := url.Parse("https://central:443")
		require.NoError(t, err)

		h := newTestHandler(t, baseURL, mockTransport, newUnauthenticatedAuthorizer(t), "test-token")
		h.centralReachable.Store(true)

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.False(t, proxyCalled, "proxy should not be called with unauthenticated token")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
