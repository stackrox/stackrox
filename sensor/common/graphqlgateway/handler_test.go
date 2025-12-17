package graphqlgateway

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
)

// mockTokenManager is a mock implementation of the TokenManager for testing
type mockTokenManager struct {
	getTokenFunc func(ctx context.Context, bearerToken, namespace, deployment string) (string, error)
}

func (m *mockTokenManager) GetToken(ctx context.Context, bearerToken, namespace, deployment string) (string, error) {
	if m.getTokenFunc != nil {
		return m.getTokenFunc(ctx, bearerToken, namespace, deployment)
	}
	return "mock-scoped-token", nil
}

func TestHandler_ServeHTTP_MethodValidation(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		statusCode int
	}{
		{
			name:       "POST method allowed",
			method:     http.MethodPost,
			statusCode: http.StatusOK,
		},
		{
			name:       "GET method rejected",
			method:     http.MethodGet,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "PUT method rejected",
			method:     http.MethodPut,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "DELETE method rejected",
			method:     http.MethodDelete,
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{
				centralClient: &http.Client{
					Transport: httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(bytes.NewBufferString(`{"data": {}}`)),
						}, nil
					}),
				},
				tokenManager: &mockTokenManager{},
			}

			req := httptest.NewRequest(tt.method, "/api/graphql-gateway", bytes.NewBufferString(`{"query": "{ deployments { name } }"}`))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("X-Namespace", "default")
			req.Header.Set("X-Deployment", "nginx")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.statusCode, w.Code)
		})
	}
}

func TestHandler_ServeHTTP_AuthorizationHeader(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		wantError  bool
		statusCode int
	}{
		{
			name:       "valid Bearer token",
			authHeader: "Bearer valid-token-12345",
			wantError:  false,
			statusCode: http.StatusOK,
		},
		{
			name:       "missing Authorization header",
			authHeader: "",
			wantError:  true,
			statusCode: http.StatusUnauthorized,
		},
		{
			name:       "invalid Bearer format - missing Bearer prefix",
			authHeader: "valid-token-12345",
			wantError:  true,
			statusCode: http.StatusUnauthorized,
		},
		{
			name:       "invalid Bearer format - wrong prefix",
			authHeader: "Basic dXNlcjpwYXNz",
			wantError:  true,
			statusCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{
				centralClient: &http.Client{
					Transport: httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(bytes.NewBufferString(`{"data": {}}`)),
						}, nil
					}),
				},
				tokenManager: &mockTokenManager{},
			}

			req := httptest.NewRequest(http.MethodPost, "/api/graphql-gateway", bytes.NewBufferString(`{"query": "{ deployments { name } }"}`))
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			req.Header.Set("X-Namespace", "default")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.statusCode, w.Code)
			if tt.wantError {
				// Should have error response
				assert.NotEmpty(t, w.Body.String())
			}
		})
	}
}

func TestHandler_ServeHTTP_HeaderExtraction(t *testing.T) {
	var capturedNamespace, capturedDeployment string

	handler := &Handler{
		centralClient: &http.Client{
			Transport: httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"data": {}}`)),
				}, nil
			}),
		},
		tokenManager: &mockTokenManager{
			getTokenFunc: func(ctx context.Context, bearerToken, namespace, deployment string) (string, error) {
				capturedNamespace = namespace
				capturedDeployment = deployment
				return "scoped-token", nil
			},
		},
	}

	tests := []struct {
		name               string
		namespace          string
		deployment         string
		expectedNamespace  string
		expectedDeployment string
	}{
		{
			name:               "both namespace and deployment specified",
			namespace:          "production",
			deployment:         "api-server",
			expectedNamespace:  "production",
			expectedDeployment: "api-server",
		},
		{
			name:               "only namespace specified",
			namespace:          "staging",
			deployment:         "",
			expectedNamespace:  "staging",
			expectedDeployment: "",
		},
		{
			name:               "no headers specified",
			namespace:          "",
			deployment:         "",
			expectedNamespace:  "",
			expectedDeployment: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capturedNamespace = ""
			capturedDeployment = ""

			req := httptest.NewRequest(http.MethodPost, "/api/graphql-gateway", bytes.NewBufferString(`{"query": "{ deployments { name } }"}`))
			req.Header.Set("Authorization", "Bearer test-token")
			if tt.namespace != "" {
				req.Header.Set("X-Namespace", tt.namespace)
			}
			if tt.deployment != "" {
				req.Header.Set("X-Deployment", tt.deployment)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.expectedNamespace, capturedNamespace)
			assert.Equal(t, tt.expectedDeployment, capturedDeployment)
		})
	}
}

func TestHandler_ServeHTTP_TokenManagerErrors(t *testing.T) {
	tests := []struct {
		name          string
		tokenError    error
		expectedCode  int
		errorContains string
	}{
		{
			name:          "invalid credentials error",
			tokenError:    errox.NoCredentials.New("invalid credentials"),
			expectedCode:  http.StatusUnauthorized,
			errorContains: "credentials",
		},
		{
			name:          "permission denied error",
			tokenError:    errox.NotAuthorized.New("not authorized to access deployment"),
			expectedCode:  http.StatusForbidden,
			errorContains: "not authorized",
		},
		{
			name:          "central offline error",
			tokenError:    errox.ServerError.New("Central is offline and no cached token available"),
			expectedCode:  http.StatusServiceUnavailable,
			errorContains: "offline",
		},
		{
			name:          "generic server error",
			tokenError:    errors.New("internal server error"),
			expectedCode:  http.StatusInternalServerError,
			errorContains: "authorization failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{
				centralClient: &http.Client{},
				tokenManager: &mockTokenManager{
					getTokenFunc: func(ctx context.Context, bearerToken, namespace, deployment string) (string, error) {
						return "", tt.tokenError
					},
				},
			}

			req := httptest.NewRequest(http.MethodPost, "/api/graphql-gateway", bytes.NewBufferString(`{"query": "{ deployments { name } }"}`))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("X-Namespace", "default")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
			assert.Contains(t, w.Body.String(), tt.errorContains)
		})
	}
}

func TestHandler_ServeHTTP_GraphQLProxying(t *testing.T) {
	var capturedAuthHeader string
	var capturedBody string
	expectedResponse := `{"data": {"deployments": [{"name": "nginx"}]}}`

	handler := &Handler{
		centralClient: &http.Client{
			Transport: httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				// Capture the Authorization header sent to Central
				capturedAuthHeader = req.Header.Get("Authorization")

				// Capture the request body
				if req.Body != nil {
					bodyBytes, _ := io.ReadAll(req.Body)
					capturedBody = string(bodyBytes)
				}

				// Verify path is correct
				assert.Equal(t, "/api/graphql", req.URL.Path)

				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(bytes.NewBufferString(expectedResponse)),
				}, nil
			}),
		},
		tokenManager: &mockTokenManager{
			getTokenFunc: func(ctx context.Context, bearerToken, namespace, deployment string) (string, error) {
				return "scoped-token-abc123", nil
			},
		},
	}

	requestBody := `{"query": "{ deployments { name } }"}`
	req := httptest.NewRequest(http.MethodPost, "/api/graphql-gateway", bytes.NewBufferString(requestBody))
	req.Header.Set("Authorization", "Bearer ocp-token")
	req.Header.Set("X-Namespace", "default")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, expectedResponse, w.Body.String())

	// Verify scoped token was used in Authorization header
	assert.Equal(t, "Bearer scoped-token-abc123", capturedAuthHeader)

	// Verify request body was proxied correctly
	assert.JSONEq(t, requestBody, capturedBody)
}

func TestHandler_ServeHTTP_TraceIDHeader(t *testing.T) {
	handler := &Handler{
		centralClient: &http.Client{
			Transport: httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"data": {}}`)),
				}, nil
			}),
		},
		tokenManager: &mockTokenManager{},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/graphql-gateway", bytes.NewBufferString(`{"query": "{ deployments { name } }"}`))
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Verify trace ID header is set in response
	traceID := w.Header().Get("X-Trace-ID")
	assert.NotEmpty(t, traceID, "X-Trace-ID header should be set")

	// Trace ID should be a UUID (36 characters with hyphens)
	assert.Len(t, traceID, 36)
}

func TestHandler_ServeHTTP_ResponseStreaming(t *testing.T) {
	expectedBody := `{"data": {"deployments": [{"name": "nginx"}, {"name": "postgres"}]}}`
	expectedHeaders := http.Header{
		"Content-Type":   []string{"application/json"},
		"Cache-Control":  []string{"no-cache"},
		"Custom-Header":  []string{"test-value"},
	}

	handler := &Handler{
		centralClient: &http.Client{
			Transport: httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     expectedHeaders,
					Body:       io.NopCloser(bytes.NewBufferString(expectedBody)),
				}, nil
			}),
		},
		tokenManager: &mockTokenManager{},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/graphql-gateway", bytes.NewBufferString(`{"query": "{ deployments { name } }"}`))
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Verify status code
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify response body
	assert.JSONEq(t, expectedBody, w.Body.String())

	// Verify headers are proxied (excluding trace ID)
	for key, values := range expectedHeaders {
		assert.Equal(t, values, w.Header().Values(key), "Header %s should be proxied", key)
	}
}

func TestHandler_ServeHTTP_CentralHTTPErrors(t *testing.T) {
	handler := &Handler{
		centralClient: &http.Client{
			Transport: httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("connection refused")
			}),
		},
		tokenManager: &mockTokenManager{},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/graphql-gateway", bytes.NewBufferString(`{"query": "{ deployments { name } }"}`))
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Should return internal server error
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to contact central")
}

func TestHandler_Notify(t *testing.T) {
	handler := &Handler{
		centralClient: &http.Client{},
		tokenManager:  &mockTokenManager{},
	}

	// Test offline notification
	handler.Notify(common.SensorComponentEventOfflineMode)
	assert.False(t, handler.centralReachable.Load())

	// Test online notification
	handler.Notify(common.SensorComponentEventCentralReachable)
	assert.True(t, handler.centralReachable.Load())
}

func TestNewHandler(t *testing.T) {
	t.Skip("NewHandler requires actual Central endpoint and certificates - test in integration tests")
}

// fakeK8sClient is a minimal fake for testing NewHandler
type fakeK8sClient struct {
	kubernetes.Interface
}
