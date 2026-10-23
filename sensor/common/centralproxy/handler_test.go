package centralproxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	apiV1 "github.com/stackrox/rox/generated/api/v1"
	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/env"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/pods"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/centralproxy/allowedpaths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"
)

func TestPolicyHandlerEnforcesIdentityAndRoutes(t *testing.T) {
	client := fake.NewClientset()
	policy := NewClientPolicy(client)
	require.NoError(t, policy.EnsureDefaults(context.Background()))
	doc := defaultClientPolicy()
	doc.Clients = append(doc.Clients, clientPolicyEntry{
		AuthType: clientAuthKubernetesBearer, Subject: "test-user", PermissionSetID: "custom-read-only-set",
		HTTPScopes: tokens.HTTPRequestScopes{"/v1/custom": {"GET"}},
	})
	data, err := json.Marshal(doc)
	require.NoError(t, err)
	cm, err := client.CoreV1().ConfigMaps(pods.GetPodNamespace()).Get(context.Background(), ClientPolicyConfigMapName, metav1.GetOptions{})
	require.NoError(t, err)
	cm.Data[ClientPolicyDataKey] = string(data)
	_, err = client.CoreV1().ConfigMaps(cm.Namespace).Update(context.Background(), cm, metav1.UpdateOptions{})
	require.NoError(t, err)
	client.PrependReactor("create", "tokenreviews", func(k8sTesting.Action) (bool, runtime.Object, error) {
		return true, &authenticationv1.TokenReview{Status: authenticationv1.TokenReviewStatus{
			Authenticated: true, User: authenticationv1.UserInfo{Username: "test-user"},
		}}, nil
	})
	client.PrependReactor("create", "subjectaccessreviews", func(k8sTesting.Action) (bool, runtime.Object, error) {
		return true, &authv1.SubjectAccessReview{Status: authv1.SubjectAccessReviewStatus{Allowed: true}}, nil
	})
	centralcaps.Set([]centralsensor.CentralCapability{
		centralsensor.InternalTokenAPISupported, centralsensor.CentralProxyReadOnlyProfiles,
	})
	t.Cleanup(func() { centralcaps.Set(nil) })
	var forwarded bool
	baseURL, err := url.Parse("https://central:443")
	require.NoError(t, err)
	var requester string
	base := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		forwarded = true
		assert.Equal(t, "Bearer sensor-token", req.Header.Get("Authorization"))
		assert.Empty(t, req.Header.Get(stackroxNamespaceHeader))
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})
	tokenClient := &fakeTokenServiceClient{
		response: &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "sensor-token"},
	}
	provider := newTokenProvider(&fakeClusterIDGetter{clusterID: "sensor-cluster"})
	var serviceClient centralv1.TokenServiceClient = tokenClient
	provider.client.Store(&serviceClient)
	h := &Handler{
		policy: policy, authorizer: newK8sAuthorizer(client), metricsEnabled: true,
		proxy: newReverseProxyForTest(baseURL, &scopedTokenTransport{base: base, tokenProvider: provider}),
	}
	h.centralReachable.Store(true)
	request := func(method, path string) *http.Request {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer kube-token")
		req.Header.Set(stackroxNamespaceHeader, "test-namespace")
		return req
	}
	for name, tc := range map[string]struct {
		method, path string
		wantCode     int
	}{
		"permitted":      {http.MethodGet, "/v1/custom", http.StatusOK},
		"wrong method":   {http.MethodPost, "/v1/custom", http.StatusForbidden},
		"unlisted route": {http.MethodGet, "/v1/private", http.StatusForbidden},
		"encoded route":  {http.MethodGet, "/v1/%63ustom", http.StatusForbidden},
	} {
		t.Run(name, func(t *testing.T) {
			forwarded = false
			w := httptest.NewRecorder()
			h.ServeHTTP(w, request(tc.method, tc.path))
			assert.Equal(t, tc.wantCode, w.Code)
			assert.Equal(t, tc.wantCode == http.StatusOK, forwarded)
			if forwarded {
				requester = tokenClient.lastRequest.GetRequester()
				assert.Equal(t, "custom-read-only-set", tokenClient.lastRequest.GetPermissionSetId())
				assert.Equal(t, "sensor-cluster", tokenClient.lastRequest.GetClusterScopes()[0].GetClusterId())
				assert.Equal(t, []string{"test-namespace"}, tokenClient.lastRequest.GetClusterScopes()[0].GetNamespaces())
			}
		})
	}
	assert.Equal(t, "kubernetesBearer:test-user", requester)
}

func TestLegacyProxyGrantAllowed(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.CentralProxyPathFiltering})
	allowedpaths.Set([]string{"/metrics", "/v1/metadata"})
	t.Cleanup(func() { centralcaps.Set(nil); allowedpaths.Reset() })
	for name, tc := range map[string]struct {
		permissionSet string
		path          string
		method        string
		allowed       bool
	}{
		"console default": {centralproxy.ConsoleProxyPermissionSetID, "/v1/metadata", "GET", true},
		"metrics default": {centralproxy.MetricsPermissionSetID, "/metrics", "GET", true},
		"generic set":     {"custom-read-only-set", "/v1/metadata", "GET", false},
		"extra method":    {centralproxy.MetricsPermissionSetID, "/metrics", "POST", false},
		"extra route":     {centralproxy.ConsoleProxyPermissionSetID, "/v1/private", "GET", false},
	} {
		t.Run(name, func(t *testing.T) {
			grant := &ClientPolicyGrant{PermissionSetID: tc.permissionSet,
				HTTPRequestScopes: []*apiV1.HTTPRequestScope{{Path: tc.path, Methods: []string{tc.method}}}}
			assert.Equal(t, tc.allowed, legacyProxyGrantAllowed(grant, httptest.NewRequest(tc.method, tc.path, nil)))
		})
	}
}

func TestPolicyHandlerVerifiesDefaultMetricsClient(t *testing.T) {
	t.Setenv("ROX_CENTRAL_PROXY_ENABLED", "true")
	ca, caKey, caPEM := clientPolicyTestCA(t)
	client := fake.NewClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: env.SecureMetricsClientCANamespace.Setting(),
			Name:      env.SecureMetricsClientCAConfigMap.Setting(),
		},
		Data: map[string]string{env.SecureMetricsClientCAKey.Setting(): caPEM},
	})
	policy := NewClientPolicy(client)
	require.NoError(t, policy.EnsureDefaults(context.Background()))
	centralcaps.Set([]centralsensor.CentralCapability{
		centralsensor.InternalTokenAPISupported, centralsensor.CentralProxyReadOnlyProfiles,
	})
	t.Cleanup(func() { centralcaps.Set(nil) })
	var forwarded bool
	baseURL, err := url.Parse("https://central:443")
	require.NoError(t, err)
	base := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		forwarded = true
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})
	tokenClient := &fakeTokenServiceClient{
		response: &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "metrics-token"},
	}
	provider := newTokenProvider(&fakeClusterIDGetter{clusterID: "sensor-cluster"})
	var serviceClient centralv1.TokenServiceClient = tokenClient
	provider.client.Store(&serviceClient)
	h := &Handler{policy: policy, authorizer: newK8sAuthorizer(client),
		proxy: newReverseProxyForTest(baseURL, &scopedTokenTransport{base: base, tokenProvider: provider})}
	h.centralReachable.Store(true)
	cert := signedTestCertificate(t, ca, caKey, env.SecureMetricsClientCertCN.Setting(), x509.ExtKeyUsageClientAuth)
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{leaf}}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, forwarded)
	assert.Equal(t, "mtls:"+leaf.Subject.CommonName, tokenClient.lastRequest.GetRequester())
	assert.Equal(t, centralproxy.MetricsPermissionSetID, tokenClient.lastRequest.GetPermissionSetId())
	assert.Empty(t, tokenClient.lastRequest.GetClusterScopes())
	assert.Equal(t, []*apiV1.HTTPRequestScope{{Path: "/metrics", Methods: []string{"GET"}}},
		tokenClient.lastRequest.GetHttpRequestScopes())

	otherCA, otherKey, _ := clientPolicyTestCA(t)
	badCert := signedTestCertificate(t, otherCA, otherKey, env.SecureMetricsClientCertCN.Setting(), x509.ExtKeyUsageClientAuth)
	badLeaf, err := x509.ParseCertificate(badCert.Certificate[0])
	require.NoError(t, err)
	forwarded = false
	req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{badLeaf}}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, forwarded)
}

func TestPolicyHandlerDoesNotEnableConsoleWhenOnlyMetricsEnabled(t *testing.T) {
	t.Setenv("ROX_OCP_CONSOLE_INTEGRATION", "false")
	client := fake.NewClientset()
	policy := NewClientPolicy(client)
	require.NoError(t, policy.EnsureDefaults(context.Background()))
	client.PrependReactor("create", "tokenreviews", func(k8sTesting.Action) (bool, runtime.Object, error) {
		return true, &authenticationv1.TokenReview{Status: authenticationv1.TokenReviewStatus{
			Authenticated: true, User: authenticationv1.UserInfo{Username: "test-user"},
		}}, nil
	})
	centralcaps.Set([]centralsensor.CentralCapability{
		centralsensor.InternalTokenAPISupported, centralsensor.CentralProxyReadOnlyProfiles,
	})
	t.Cleanup(func() { centralcaps.Set(nil) })
	h := &Handler{policy: policy, authorizer: newK8sAuthorizer(client), metricsEnabled: true}
	h.centralReachable.Store(true)
	req := httptest.NewRequest(http.MethodGet, "/v1/metadata", nil)
	req.Header.Set("Authorization", "Bearer kube-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestPolicyHandlerAuthenticatesServiceAccountBearerWithoutCA(t *testing.T) {
	t.Setenv("ROX_CENTRAL_PROXY_ENABLED", "true")
	client := fake.NewClientset()
	policy := NewClientPolicy(client)
	require.NoError(t, policy.EnsureDefaults(context.Background()))
	doc := defaultClientPolicy()
	doc.Clients = append(doc.Clients, clientPolicyEntry{
		AuthType:        clientAuthKubernetesBearer,
		Subject:         "system:serviceaccount:observability:scraper",
		PermissionSetID: centralproxy.MetricsPermissionSetID,
		HTTPScopes:      tokens.HTTPRequestScopes{"/metrics": {"GET"}},
	})
	setClientPolicyConfigMap(t, context.Background(), client, doc)
	client.PrependReactor("create", "tokenreviews", func(k8sTesting.Action) (bool, runtime.Object, error) {
		return true, &authenticationv1.TokenReview{Status: authenticationv1.TokenReviewStatus{
			Authenticated: true,
			User:          authenticationv1.UserInfo{Username: "system:serviceaccount:observability:scraper"},
		}}, nil
	})
	centralcaps.Set([]centralsensor.CentralCapability{
		centralsensor.InternalTokenAPISupported, centralsensor.CentralProxyReadOnlyProfiles,
	})
	t.Cleanup(func() { centralcaps.Set(nil) })
	baseURL, err := url.Parse("https://central:443")
	require.NoError(t, err)
	tokenClient := &fakeTokenServiceClient{
		response: &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "sensor-token"},
	}
	provider := newTokenProvider(&fakeClusterIDGetter{clusterID: "sensor-cluster"})
	var serviceClient centralv1.TokenServiceClient = tokenClient
	provider.client.Store(&serviceClient)
	var forwarded bool
	base := pkghttputil.RoundTripperFunc(func(*http.Request) (*http.Response, error) {
		forwarded = true
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})
	h := &Handler{policy: policy, authorizer: newK8sAuthorizer(client),
		proxy: newReverseProxyForTest(baseURL, &scopedTokenTransport{base: base, tokenProvider: provider})}
	h.centralReachable.Store(true)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer kube-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, forwarded)
	assert.Equal(t, "kubernetesBearer:system:serviceaccount:observability:scraper", tokenClient.lastRequest.GetRequester())
	assert.Empty(t, tokenClient.lastRequest.GetClusterScopes())
}

func TestPolicyHandlerPreservesBuiltInClientOnOlderCentral(t *testing.T) {
	t.Setenv("ROX_OCP_CONSOLE_INTEGRATION", "true")
	client := fake.NewClientset()
	policy := NewClientPolicy(client)
	require.NoError(t, policy.EnsureDefaults(context.Background()))
	f := newProxyTestFixture(t, newAllowingAuthorizer(t))
	f.handler.policy = policy
	centralcaps.Set([]centralsensor.CentralCapability{
		centralsensor.InternalTokenAPISupported, centralsensor.CentralProxyPathFiltering,
	})
	allowedpaths.Set([]string{"/v1/metadata"})
	w := f.serveHTTP(t, http.MethodGet, "/v1/metadata", map[string]string{"Authorization": "Bearer kube-token"})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, f.proxyCalled)

	doc := defaultClientPolicy()
	doc.Clients = append(doc.Clients, clientPolicyEntry{
		AuthType: clientAuthKubernetesBearer, Subject: "test-user", PermissionSetID: "custom-read-only-set",
		HTTPScopes: tokens.HTTPRequestScopes{"/v1/custom": {"GET"}},
	})
	setClientPolicyConfigMap(t, context.Background(), client, doc)
	f.proxyCalled = false
	w = f.serveHTTP(t, http.MethodGet, "/v1/custom", map[string]string{"Authorization": "Bearer kube-token"})
	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.False(t, f.proxyCalled)
}

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

func TestProxyPathFilteringCompatibility(t *testing.T) {
	for name, tc := range map[string]struct {
		advertisesFiltering bool
		allowedPaths        []string
		path                string
		allowed             bool
	}{
		"older Central without path filtering": {
			allowedPaths: []string{"/v1/deployments"}, path: "/v1/alerts", allowed: true,
		},
		"advertised path is allowed": {
			advertisesFiltering: true, allowedPaths: []string{"/v1/deployments"},
			path: "/v1/deployments", allowed: true,
		},
		"unadvertised path is denied": {
			advertisesFiltering: true, allowedPaths: []string{"/v1/deployments"}, path: "/v1/alerts",
		},
		"empty advertised list denies paths": {
			advertisesFiltering: true, path: "/v1/alerts",
		},
	} {
		t.Run(name, func(t *testing.T) {
			centralcaps.Set(nil)
			if tc.advertisesFiltering {
				centralcaps.Set([]centralsensor.CentralCapability{centralsensor.CentralProxyPathFiltering})
			}
			allowedpaths.Set(tc.allowedPaths)
			t.Cleanup(func() {
				centralcaps.Set(nil)
				allowedpaths.Reset()
			})

			h := &Handler{}
			h.centralReachable.Store(true)
			err := h.validateRequest(httptest.NewRequest(http.MethodGet, tc.path, nil))
			if tc.allowed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, http.StatusForbidden, pkghttputil.StatusFromError(err))
			}
		})
	}
}

func TestMetricsScrapeRoute(t *testing.T) {
	for _, tc := range []struct {
		name     string
		path     string
		method   string
		cn       string
		optIn    bool
		paths    []string
		wantCode int
	}{
		{"disabled by default", "/metrics", http.MethodGet, env.SecureMetricsClientCertCN.Setting(), false,
			[]string{"/metrics"}, http.StatusForbidden},
		{"missing client certificate", "/metrics", http.MethodGet, "", true,
			[]string{"/metrics"}, http.StatusUnauthorized},
		{"wrong client identity", "/metrics", http.MethodGet, "other", true,
			[]string{"/metrics"}, http.StatusUnauthorized},
		{"legacy Central without allowlist", "/metrics", http.MethodGet, env.SecureMetricsClientCertCN.Setting(), true,
			nil, http.StatusForbidden},
		{"POST rejected", "/metrics", http.MethodPost, env.SecureMetricsClientCertCN.Setting(), true,
			[]string{"/metrics"}, http.StatusForbidden},
		{"trailing slash rejected", "/metrics/", http.MethodGet, env.SecureMetricsClientCertCN.Setting(), true,
			[]string{"/metrics"}, http.StatusForbidden},
		{"encoded path rejected", "/m%65trics", http.MethodGet, env.SecureMetricsClientCertCN.Setting(), true,
			[]string{"/metrics"}, http.StatusForbidden},
		{"doubly encoded path rejected", "/m%2565trics", http.MethodGet, env.SecureMetricsClientCertCN.Setting(), true,
			[]string{"/metrics"}, http.StatusForbidden},
		{"triply encoded path rejected", "/m%252565trics", http.MethodGet, env.SecureMetricsClientCertCN.Setting(), true,
			[]string{"/metrics"}, http.StatusForbidden},
		{"verified metrics scrape", "/metrics", http.MethodGet, env.SecureMetricsClientCertCN.Setting(), true,
			[]string{"/metrics"}, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newProxyTestFixture(t, newDenyingAuthorizer(t))
			f.handler.metricsEnabled = tc.optIn
			allowedpaths.Set(tc.paths)
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.cn != "" {
				req.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{
					Subject: pkix.Name{CommonName: tc.cn},
				}}}}
			}
			w := httptest.NewRecorder()
			f.handler.ServeHTTP(w, req)
			assert.Equal(t, tc.wantCode, w.Code)
			assert.Equal(t, tc.wantCode == http.StatusOK, f.proxyCalled)
		})
	}
}

func TestMetricsOptInDoesNotEnableConsoleProxy(t *testing.T) {
	t.Setenv("ROX_OCP_CONSOLE_INTEGRATION", "false")
	f := newProxyTestFixture(t, newDenyingAuthorizer(t))
	f.handler.metricsEnabled = true
	allowedpaths.Set([]string{"/metrics", "/api/graphql"})

	w := f.serveHTTP(t, http.MethodGet, "/api/graphql", map[string]string{"Authorization": "Bearer console-token"})
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, f.proxyCalled)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{
		Subject: pkix.Name{CommonName: env.SecureMetricsClientCertCN.Setting()},
	}}}}
	w = httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, f.proxyCalled)
}

func TestServeHTTP(t *testing.T) {
	t.Run("validation fails, proxy not called", func(t *testing.T) {
		setupCentralCapsForTest(t)
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
		f := newProxyTestFixture(t, newAllowingAuthorizer(t))
		w := f.serveHTTP(t, http.MethodGet, "/v1/alerts", map[string]string{
			"Authorization": "Bearer test-token",
		})
		assert.True(t, f.proxyCalled, "proxy should be called")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("proxy error handled by ErrorHandler", func(t *testing.T) {
		setupCentralCapsForTest(t)

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

	t.Run("request rejected when Central lacks internal token API capability", func(t *testing.T) {
		// Explicitly clear central caps to simulate an older Central without the capability.
		// Use cleanup to restore state and avoid cross-test interference.
		centralcaps.Set(nil)
		t.Cleanup(func() {
			centralcaps.Set(nil)
		})

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

		h := &Handler{
			proxy:      newReverseProxyForTest(baseURL, mockTransport),
			authorizer: newAllowingAuthorizer(t),
		}
		h.centralReachable.Store(true)

		req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		assert.False(t, proxyCalled, "proxy should not be called when capability is missing")
		assert.Equal(t, http.StatusNotImplemented, w.Code)
		assert.Contains(t, w.Body.String(), "proxy to Central is not available")
		assert.Contains(t, w.Body.String(), "internal token API")
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
			setupCentralCapsForTest(t)
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
		f := newProxyTestFixture(t, newDenyingAuthorizer(t))
		// Set namespace scope header to trigger SAR check.
		w := f.serveHTTP(t, http.MethodGet, "/v1/alerts", map[string]string{
			"Authorization":         "Bearer test-token",
			stackroxNamespaceHeader: "my-namespace",
		})
		assert.False(t, f.proxyCalled, "proxy should not be called when authorization fails")
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "lacks")
	})

	t.Run("no authorizer returns server error", func(t *testing.T) {
		f := newProxyTestFixture(t, nil)
		w := f.serveHTTP(t, http.MethodGet, "/v1/alerts", nil)
		assert.False(t, f.proxyCalled, "proxy should not be called when authorizer is not configured")
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "authorizer not configured")
	})

	t.Run("authorization success allows proxy call", func(t *testing.T) {
		f := newProxyTestFixture(t, newAllowingAuthorizer(t))
		w := f.serveHTTP(t, http.MethodGet, "/v1/alerts", map[string]string{
			"Authorization": "Bearer test-token",
		})
		assert.True(t, f.proxyCalled, "proxy should be called when authorization succeeds")
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestServeHTTP_NamespaceScopeBasedAuthorization(t *testing.T) {
	t.Run("empty namespace scope skips SAR check", func(t *testing.T) {
		// Create authorizer that denies SAR but allows TokenReview.
		f := newProxyTestFixture(t, newDenyingAuthorizer(t))
		// No namespace scope header - should skip SAR.
		w := f.serveHTTP(t, http.MethodGet, "/v1/metadata", map[string]string{
			"Authorization": "Bearer test-token",
		})
		assert.True(t, f.proxyCalled, "proxy should be called when namespace scope is empty (no SAR)")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("specific namespace scope triggers SAR check", func(t *testing.T) {
		f := newProxyTestFixture(t, newDenyingAuthorizer(t))
		w := f.serveHTTP(t, http.MethodGet, "/v1/alerts", map[string]string{
			"Authorization":         "Bearer test-token",
			stackroxNamespaceHeader: "my-namespace",
		})
		assert.False(t, f.proxyCalled, "proxy should not be called when SAR fails for namespace scope")
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("cluster-wide scope (*) triggers SAR check", func(t *testing.T) {
		f := newProxyTestFixture(t, newDenyingAuthorizer(t))
		w := f.serveHTTP(t, http.MethodGet, "/v1/alerts", map[string]string{
			"Authorization":         "Bearer test-token",
			stackroxNamespaceHeader: FullClusterAccessScope,
		})
		assert.False(t, f.proxyCalled, "proxy should not be called when SAR fails for cluster-wide scope")
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("namespace scope with valid permissions succeeds", func(t *testing.T) {
		setupCentralCapsForTest(t)
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
		setupCentralCapsForTest(t)
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
		setupCentralCapsForTest(t)
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
		setupCentralCapsForTest(t)
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
		f := newProxyTestFixture(t, newAllowingAuthorizer(t))
		// No Authorization header set.
		w := f.serveHTTP(t, http.MethodGet, "/v1/alerts", nil)
		assert.False(t, f.proxyCalled, "proxy should not be called without authorization header")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "bearer token")
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		f := newProxyTestFixture(t, newUnauthenticatedAuthorizer(t))
		w := f.serveHTTP(t, http.MethodGet, "/v1/alerts", map[string]string{
			"Authorization": "Bearer invalid-token",
		})
		assert.False(t, f.proxyCalled, "proxy should not be called with unauthenticated token")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestServeHTTP_PathFiltering(t *testing.T) {
	t.Run("disallowed path returns 403", func(t *testing.T) {
		f := newProxyTestFixture(t, newAllowingAuthorizer(t))
		w := f.serveHTTP(t, http.MethodGet, "/admin/secret", map[string]string{
			"Authorization": "Bearer test-token",
		})
		assert.False(t, f.proxyCalled, "proxy should not be called for disallowed path")
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "not allowed by the proxy allow-list")
	})

	t.Run("allowed path proceeds normally", func(t *testing.T) {
		f := newProxyTestFixture(t, newAllowingAuthorizer(t))
		w := f.serveHTTP(t, http.MethodGet, "/v1/alerts", map[string]string{
			"Authorization": "Bearer test-token",
		})
		assert.True(t, f.proxyCalled, "proxy should be called for allowed path")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("no allow-list configured allows all paths", func(t *testing.T) {
		f := newProxyTestFixture(t, newAllowingAuthorizer(t))
		// Override: remove path filtering capability and clear allowed paths.
		centralcaps.Set([]centralsensor.CentralCapability{
			centralsensor.InternalTokenAPISupported,
		})
		allowedpaths.Reset()

		w := f.serveHTTP(t, http.MethodGet, "/admin/anything", map[string]string{
			"Authorization": "Bearer test-token",
		})
		assert.True(t, f.proxyCalled, "proxy should be called when no allow-list is configured")
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
