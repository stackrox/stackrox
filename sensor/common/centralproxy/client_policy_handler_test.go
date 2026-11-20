package centralproxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"

	apiV1 "github.com/stackrox/rox/generated/api/v1"
	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/centralproxy/allowedpaths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"
)

func newPolicyHandlerFixture(t *testing.T) (*proxyTestFixture, *ClientPolicy) {
	t.Helper()
	fixture := newProxyTestFixture(t, newAllowingAuthorizer(t))
	return fixture, fixture.handler.policy
}

func TestPolicyHandlerEnforcesIdentityAndRoutes(t *testing.T) {
	f, policy := newPolicyHandlerFixture(t)
	doc := defaultClientPolicy()
	doc.Clients = append(doc.Clients, clientPolicyEntry{
		AuthType: clientAuthKubernetesBearer, Subject: "test-user",
		Access: []clientPolicyAccess{
			{PermissionSetName: "custom-read-only-set", HTTPScopes: tokens.HTTPRequestScopes{"/v1/custom": {"GET"}}},
			{PermissionSetName: "other-read-only-set", HTTPScopes: tokens.HTTPRequestScopes{"/v1/other": {"POST"}}},
		},
	})
	setClientPolicyConfigMap(t, t.Context(), policy.client.(*fake.Clientset), doc)
	waitForClientPolicy(t, policy, doc)
	var calls atomic.Int32
	client := &fakeTokenServiceClient{
		response: &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "sensor-token"}, callCount: &calls,
	}
	f.handler.transport.tokenProvider = newTestTokenProvider(client, "sensor-cluster")
	headers := map[string]string{"Authorization": "Bearer kube-token", stackroxNamespaceHeader: "test-namespace"}
	for name, tc := range map[string]struct {
		method, path string
		wantCode     int
	}{
		"permitted":            {http.MethodGet, "/v1/custom", http.StatusOK},
		"other permission set": {http.MethodPost, "/v1/other", http.StatusOK},
		"wrong method":         {http.MethodPost, "/v1/custom", http.StatusForbidden},
		"unlisted route":       {http.MethodGet, "/v1/private", http.StatusForbidden},
		"encoded route":        {http.MethodGet, "/v1/%63ustom", http.StatusForbidden},
	} {
		t.Run(name, func(t *testing.T) {
			f.proxyCalled = false
			w := f.serveHTTP(t, tc.method, tc.path, headers)
			assert.Equal(t, tc.wantCode, w.Code)
			assert.Equal(t, tc.wantCode == http.StatusOK, f.proxyCalled)
			if f.proxyCalled {
				assert.Equal(t, "kubernetesBearer:test-user", client.lastRequest.GetRequester())
				protoassert.SlicesEqual(t, clientPolicyAccessToProto(doc.Clients[2].Access), client.lastRequest.GetPermissionSets())
				protoassert.SlicesEqual(t, []*centralv1.ClusterScope{{ClusterId: "sensor-cluster", Namespaces: []string{"test-namespace"}}},
					client.lastRequest.GetClusterScopes())
			}
		})
	}
	assert.EqualValues(t, 1, calls.Load(), "both routes reuse the same profile token")
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.InternalTokenAPISupported})
	w := f.serveHTTP(t, http.MethodGet, "/v1/custom", headers)
	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.EqualValues(t, 1, calls.Load())
}

func TestPolicyHandlerMixedGrantsDoNotBypassCustomClientChecks(t *testing.T) {
	for order, reverse := range map[string]bool{"console first": false, "custom first": true} {
		t.Run(order, func(t *testing.T) {
			access := []clientPolicyAccess{
				{PermissionSetName: accesscontrol.K8sCentralProxyUser, HTTPScopes: consolePolicyHTTPScopes()},
				{PermissionSetName: "Custom Reader"},
			}
			if reverse {
				slices.Reverse(access)
			}
			for name, tc := range map[string]struct {
				consoleEnabled, perAccess, sarAllowed bool
				namespace                             string
				wantStatus                            int
			}{
				"custom works with console disabled": {perAccess: true, sarAllowed: true, namespace: "team-a", wantStatus: http.StatusOK},
				"namespace required":                 {consoleEnabled: true, perAccess: true, sarAllowed: true, wantStatus: http.StatusForbidden},
				"SAR required":                       {consoleEnabled: true, perAccess: true, namespace: "team-a", wantStatus: http.StatusForbidden},
				"no legacy fallback":                 {consoleEnabled: true, sarAllowed: true, namespace: "team-a", wantStatus: http.StatusNotImplemented},
			} {
				t.Run(name, func(t *testing.T) {
					t.Setenv("ROX_OCP_CONSOLE_INTEGRATION", fmt.Sprint(tc.consoleEnabled))
					authorizer := newDenyingAuthorizer(t)
					if tc.sarAllowed {
						authorizer = newAllowingAuthorizer(t)
					}
					fixture := newProxyTestFixture(t, authorizer)
					policy := &ClientPolicy{}
					policy.ready.Store(true)
					policy.policy.Store(&clientPolicySnapshot{document: &clientPolicyDocument{
						Version: clientPolicyVersion, Clients: []clientPolicyEntry{{
							AuthType: clientAuthKubernetesBearer, Subject: "test-user", Access: access,
						}},
					}})
					fixture.handler.policy = policy
					if !tc.perAccess {
						centralcaps.Set([]centralsensor.CentralCapability{centralsensor.InternalTokenAPISupported})
					}
					response := fixture.serveHTTP(t, http.MethodGet, "/v1/metadata", map[string]string{
						"Authorization": "Bearer kube-token", stackroxNamespaceHeader: tc.namespace,
					})
					assert.Equal(t, tc.wantStatus, response.Code)
					assert.Equal(t, tc.wantStatus == http.StatusOK, fixture.proxyCalled)
				})
			}
		})
	}
}

func TestPolicyHandlerVerifiesDefaultMetricsClient(t *testing.T) {
	ca, caKey, caPEM := clientPolicyTestCA(t)
	client := fake.NewClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: env.SecureMetricsClientCANamespace.Setting(),
			Name:      env.SecureMetricsClientCAConfigMap.Setting(),
		},
		Data: map[string]string{env.SecureMetricsClientCAKey.Setting(): caPEM},
	})
	policy := NewClientPolicy(client)
	t.Cleanup(policy.Stop)
	require.NoError(t, policy.EnsureDefaults(context.Background()))
	f := newProxyTestFixture(t, nil)
	f.handler.policy = policy
	tokenClient := &fakeTokenServiceClient{
		response: &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "metrics-token"},
	}
	f.handler.transport.tokenProvider = newTestTokenProvider(tokenClient, "sensor-cluster")
	cert := signedTestCertificate(t, ca, caKey, env.SecureMetricsClientCertCN.Setting(), x509.ExtKeyUsageClientAuth)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert.Leaf}}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, f.proxyCalled)

	f.handler.metricsEnabled = true
	w = httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, f.proxyCalled)
	assert.Equal(t, "mtls:"+cert.Leaf.Subject.CommonName, tokenClient.lastRequest.GetRequester())
	protoassert.SlicesEqual(t, []*centralv1.ClusterScope{{ClusterId: "sensor-cluster", FullClusterAccess: true}},
		tokenClient.lastRequest.GetClusterScopes())
	protoassert.SlicesEqual(t, []*centralv1.ScopedPermissionSet{{
		Selector:          &centralv1.ScopedPermissionSet_Name{Name: accesscontrol.OpenShiftPrometheus},
		HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/metrics", Methods: []string{"GET"}}},
	}}, tokenClient.lastRequest.GetPermissionSets())

	otherCA, otherKey, _ := clientPolicyTestCA(t)
	badCert := signedTestCertificate(t, otherCA, otherKey, env.SecureMetricsClientCertCN.Setting(), x509.ExtKeyUsageClientAuth)
	f.proxyCalled = false
	req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{badCert.Leaf}}
	w = httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, f.proxyCalled)

	centralcaps.Set([]centralsensor.CentralCapability{
		centralsensor.InternalTokenAPISupported, centralsensor.CentralProxyPathFiltering,
	})
	req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert.Leaf}}
	w = httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.False(t, f.proxyCalled)
}

func TestPolicyHandlerDoesNotEnableConsoleWhenOnlyMetricsEnabled(t *testing.T) {
	t.Setenv("ROX_OCP_CONSOLE_INTEGRATION", "false")
	f, _ := newPolicyHandlerFixture(t)
	f.handler.metricsEnabled = true
	w := f.serveHTTP(t, http.MethodGet, "/v1/metadata", map[string]string{"Authorization": "Bearer kube-token"})
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, f.proxyCalled)
}

func TestPolicyHandlerAuthenticatesServiceAccountBearerWithoutCA(t *testing.T) {
	f, policy := newPolicyHandlerFixture(t)
	const username = "system:serviceaccount:observability:scraper"
	f.handler.authorizer.client.(*fake.Clientset).PrependReactor("create", "tokenreviews", func(k8sTesting.Action) (bool, runtime.Object, error) {
		return true, &authenticationv1.TokenReview{Status: authenticationv1.TokenReviewStatus{
			Authenticated: true, User: authenticationv1.UserInfo{Username: username},
		}}, nil
	})
	tokenClient := &fakeTokenServiceClient{
		response: &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "sensor-token"},
	}
	f.handler.transport.tokenProvider = newTestTokenProvider(tokenClient, "sensor-cluster")
	doc := defaultClientPolicy()
	doc.Clients = append(doc.Clients, clientPolicyEntry{
		AuthType: clientAuthKubernetesBearer, Subject: username,
		Access: []clientPolicyAccess{{PermissionSetName: accesscontrol.OpenShiftPrometheus,
			HTTPScopes: tokens.HTTPRequestScopes{"/metrics": {"GET"}}}},
	})
	setClientPolicyConfigMap(t, t.Context(), policy.client.(*fake.Clientset), doc)
	waitForClientPolicy(t, policy, doc)
	headers := map[string]string{"Authorization": "Bearer kube-token"}
	w := f.serveHTTP(t, http.MethodGet, "/metrics", headers)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, f.proxyCalled)

	f.handler.metricsEnabled = true
	w = f.serveHTTP(t, http.MethodGet, "/metrics", headers)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, f.proxyCalled)

	headers[stackroxNamespaceHeader] = "observability"
	w = f.serveHTTP(t, http.MethodGet, "/metrics", headers)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, f.proxyCalled)
	assert.Equal(t, "kubernetesBearer:"+username, tokenClient.lastRequest.GetRequester())
	protoassert.SlicesEqual(t, []*centralv1.ClusterScope{{ClusterId: "sensor-cluster", Namespaces: []string{"observability"}}},
		tokenClient.lastRequest.GetClusterScopes())
}

func TestPolicyHandlerPreservesBuiltInClientOnOlderCentral(t *testing.T) {
	t.Setenv("ROX_OCP_CONSOLE_INTEGRATION", "true")
	f, policy := newPolicyHandlerFixture(t)
	centralcaps.Set([]centralsensor.CentralCapability{
		centralsensor.InternalTokenAPISupported, centralsensor.CentralProxyPathFiltering,
	})
	allowedpaths.Set([]string{"/v1/metadata"})
	w := f.serveHTTP(t, http.MethodGet, "/v1/metadata", map[string]string{"Authorization": "Bearer kube-token"})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, f.proxyCalled)

	doc := defaultClientPolicy()
	doc.Clients = append(doc.Clients, clientPolicyEntry{
		AuthType: clientAuthKubernetesBearer, Subject: "test-user",
		Access: []clientPolicyAccess{{PermissionSetName: "custom-read-only-set",
			HTTPScopes: tokens.HTTPRequestScopes{"/v1/custom": {"GET"}}}},
	})
	setClientPolicyConfigMap(t, t.Context(), policy.client.(*fake.Clientset), doc)
	waitForClientPolicy(t, policy, doc)
	f.proxyCalled = false
	w = f.serveHTTP(t, http.MethodGet, "/v1/custom", map[string]string{"Authorization": "Bearer kube-token"})
	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.False(t, f.proxyCalled)
}
