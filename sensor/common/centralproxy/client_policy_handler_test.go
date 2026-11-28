package centralproxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/centralproxy/allowedpaths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func policyHandlerFixture(t *testing.T, authorizer *k8sAuthorizer, policy *ClientPolicy) (*proxyTestFixture, *fakeTokenServiceClient) {
	t.Helper()
	fixture := newProxyTestFixture(t, authorizer)
	centralcaps.Set([]centralsensor.CentralCapability{
		centralsensor.InternalTokenAPISupported, centralsensor.CentralProxyPathFiltering, centralsensor.CentralProxyPermissionSetTokens, centralsensor.CentralProxyPerAccessTokens,
	})
	fixture.handler.policy = policy
	client := &fakeTokenServiceClient{response: &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "central-token"}}
	base := fixture.handler.proxy.Transport.(*mockTokenTransport).base
	transport := &scopedTokenTransport{base: base, tokenProvider: newTestTokenProvider(client, "sensor-cluster")}
	fixture.handler.transport = transport
	fixture.handler.proxy.Transport = transport
	return fixture, client
}

func TestCustomBearerRoutesScopeAndCapabilities(t *testing.T) {
	for name, tc := range map[string]struct {
		method     string
		namespace  string
		sarAllowed bool
		capable    bool
		want       int
	}{
		"custom GET":                      {method: http.MethodGet, namespace: "prod", sarAllowed: true, capable: true, want: http.StatusOK},
		"custom POST":                     {method: http.MethodPost, namespace: "prod", sarAllowed: true, capable: true, want: http.StatusOK},
		"custom method uses Central RBAC": {method: http.MethodDelete, namespace: "prod", sarAllowed: true, capable: true, want: http.StatusOK},
		"namespace required":              {method: http.MethodGet, sarAllowed: true, capable: true, want: http.StatusForbidden},
		"SAR required":                    {method: http.MethodGet, namespace: "prod", capable: true, want: http.StatusForbidden},
		"cluster scope requires SAR":      {method: http.MethodGet, namespace: "*", capable: true, want: http.StatusForbidden},
		"new capability required":         {method: http.MethodGet, namespace: "prod", sarAllowed: true, want: http.StatusNotImplemented},
	} {
		t.Run(name, func(t *testing.T) {
			policy := newStaticClientPolicy(t, clientPolicyDocument{Version: 1, Clients: []clientPolicyEntry{{
				AuthType: clientAuthKubernetesBearer, Subject: "test-user", Access: []clientPolicyAccess{{PermissionSetName: "Custom Reader"}},
			}}})
			authorizer := newDenyingAuthorizer(t)
			if tc.sarAllowed {
				authorizer = newAllowingAuthorizer(t)
			}
			fixture, client := policyHandlerFixture(t, authorizer, policy)
			t.Setenv("ROX_OCP_CONSOLE_INTEGRATION", "false")
			allowedpaths.Set([]string{"/v1/metadata"})
			if !tc.capable {
				centralcaps.Set([]centralsensor.CentralCapability{centralsensor.InternalTokenAPISupported, centralsensor.CentralProxyPathFiltering, centralsensor.CentralProxyPermissionSetTokens})
			}
			response := fixture.serveHTTP(t, tc.method, "/custom/route", map[string]string{
				"Authorization": "Bearer kube-token", stackroxNamespaceHeader: tc.namespace,
			})
			assert.Equal(t, tc.want, response.Code)
			assert.Equal(t, tc.want == http.StatusOK, fixture.proxyCalled)
			if tc.want == http.StatusOK {
				require.NotNil(t, client.lastRequest)
				require.Len(t, client.lastRequest.GetPermissionSets(), 1)
				assert.Equal(t, "Custom Reader", client.lastRequest.GetPermissionSets()[0].GetName())
				assert.Empty(t, client.lastRequest.GetPermissions())
				assert.Equal(t, "kubernetesBearer:test-user", client.lastRequest.GetRequester())
				require.Len(t, client.lastRequest.GetClusterScopes(), 1)
				assert.Equal(t, []string{"prod"}, client.lastRequest.GetClusterScopes()[0].GetNamespaces())
				assert.Equal(t, "sensor-cluster", client.lastRequest.GetClusterScopes()[0].GetClusterId())
			} else {
				assert.Nil(t, client.lastRequest)
			}
		})
	}
}

func TestMultiGrantClientsRequestOnlyAllowedGrants(t *testing.T) {
	policy := newStaticClientPolicy(t, clientPolicyDocument{Version: 1, Clients: []clientPolicyEntry{{
		AuthType: clientAuthKubernetesBearer, Subject: "test-user",
		Access: []clientPolicyAccess{
			{PermissionSetName: "Deployment Reader", HTTPScopes: tokens.HTTPRequestScopes{"/api/graphql": {http.MethodPost}}},
			{PermissionSetName: "Image Reader", HTTPScopes: tokens.HTTPRequestScopes{"/v1/images": {http.MethodGet}}},
		},
	}}})
	fixture, client := policyHandlerFixture(t, newAllowingAuthorizer(t), policy)
	calls := &atomic.Int32{}
	client.callCount = calls
	headers := map[string]string{"Authorization": "Bearer kube-token", stackroxNamespaceHeader: "prod"}

	response := fixture.serveHTTP(t, http.MethodPost, "/api/graphql", headers)
	assert.Equal(t, http.StatusOK, response.Code)
	require.NotNil(t, client.lastRequest)
	require.Len(t, client.lastRequest.GetPermissionSets(), 1)
	assert.Equal(t, "Deployment Reader", client.lastRequest.GetPermissionSets()[0].GetName())

	response = fixture.serveHTTP(t, http.MethodGet, "/v1/images", headers)
	assert.Equal(t, http.StatusOK, response.Code)
	require.NotNil(t, client.lastRequest)
	require.Len(t, client.lastRequest.GetPermissionSets(), 1)
	assert.Equal(t, "Image Reader", client.lastRequest.GetPermissionSets()[0].GetName())

	response = fixture.serveHTTP(t, http.MethodGet, "/v1/secrets", headers)
	assert.Equal(t, http.StatusForbidden, response.Code)
	assert.EqualValues(t, 2, calls.Load(), "routes with different allowed grants must request separate tokens")
}

func TestConsoleSelectedClientsRetainRestrictions(t *testing.T) {
	consoleID := accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy]
	for selector, id := range map[string]string{
		"name":         "",
		"ID":           consoleID,
		"uppercase ID": strings.ToUpper(consoleID),
		"braced ID":    "{" + consoleID + "}",
		"compact ID":   strings.ReplaceAll(consoleID, "-", ""),
	} {
		t.Run(selector, func(t *testing.T) {
			for name, tc := range map[string]struct {
				method          string
				path            string
				namespace       string
				consoleEnabled  bool
				legacy          bool
				want            int
				wantSets        int
				wantPermissions map[string]centralv1.Access
			}{
				"authentication-only route":                  {method: http.MethodGet, path: "/v1/metadata", consoleEnabled: true, want: http.StatusOK},
				"authentication-only route method denied":    {method: http.MethodPost, path: "/v1/metadata", consoleEnabled: true, want: http.StatusForbidden},
				"unlisted route":                             {method: http.MethodGet, path: "/custom", consoleEnabled: true, want: http.StatusForbidden},
				"data route requires namespace scope":        {method: http.MethodGet, path: "/v1/deployments", consoleEnabled: true, want: http.StatusForbidden},
				"data route with scope":                      {method: http.MethodGet, path: "/v1/deployments", namespace: "prod", consoleEnabled: true, want: http.StatusOK, wantSets: 1},
				"permissions route requires namespace scope": {method: http.MethodGet, path: "/v1/mypermissions", consoleEnabled: true, want: http.StatusForbidden},
				"permissions route with scope":               {method: http.MethodGet, path: "/v1/mypermissions", namespace: "prod", consoleEnabled: true, want: http.StatusOK, wantSets: 1},
				"console disabled":                           {method: http.MethodGet, path: "/v1/metadata", want: http.StatusForbidden},
				"legacy authentication-only route":           {method: http.MethodGet, path: "/v1/metadata", consoleEnabled: true, legacy: true, want: http.StatusOK},
				"legacy data route":                          {method: http.MethodGet, path: "/v1/deployments", namespace: "prod", consoleEnabled: true, legacy: true, want: http.StatusOK, wantPermissions: tokenPermissions},
				"legacy permissions route":                   {method: http.MethodGet, path: "/v1/mypermissions", namespace: "prod", consoleEnabled: true, legacy: true, want: http.StatusOK, wantPermissions: tokenPermissions},
			} {
				t.Run(name, func(t *testing.T) {
					entry := clientPolicyEntry{AuthType: clientAuthKubernetesBearer, Subject: "test-user",
						Access: defaultClientPolicy().Clients[1].Access}
					entry.Access[0].PermissionSetID = id
					if id == "" {
						entry.Access[0].PermissionSetName = accesscontrol.ConsoleProxy
					}
					policy := newStaticClientPolicy(t, clientPolicyDocument{Version: 1, Clients: []clientPolicyEntry{entry}})
					fixture, client := policyHandlerFixture(t, newAllowingAuthorizer(t), policy)
					t.Setenv("ROX_OCP_CONSOLE_INTEGRATION", fmt.Sprint(tc.consoleEnabled))
					allowedpaths.Set([]string{"/v1/metadata", "/v1/deployments", "/v1/mypermissions"})
					if tc.legacy {
						centralcaps.Set([]centralsensor.CentralCapability{centralsensor.InternalTokenAPISupported, centralsensor.CentralProxyPathFiltering})
					}
					headers := map[string]string{"Authorization": "Bearer kube-token"}
					if tc.namespace != "" {
						headers[stackroxNamespaceHeader] = tc.namespace
					}
					response := fixture.serveHTTP(t, tc.method, tc.path, headers)
					assert.Equal(t, tc.want, response.Code)
					assert.Equal(t, tc.want == http.StatusOK, fixture.proxyCalled)
					if tc.want == http.StatusOK {
						require.NotNil(t, client.lastRequest)
						if tc.namespace == "" {
							assert.Empty(t, client.lastRequest.GetClusterScopes(), "authentication-only bootstrap must keep an empty scope")
						} else {
							require.Len(t, client.lastRequest.GetClusterScopes(), 1)
							assert.Equal(t, []string{tc.namespace}, client.lastRequest.GetClusterScopes()[0].GetNamespaces())
						}
						require.Len(t, client.lastRequest.GetPermissionSets(), tc.wantSets)
						assert.Equal(t, tc.wantPermissions, client.lastRequest.GetPermissions())
						if tc.wantSets > 0 && id != "" {
							assert.Equal(t, consoleID, client.lastRequest.GetPermissionSets()[0].GetId())
						}
					}
				})
			}
		})
	}
}

func TestCustomMTLSRoutesAndConsoleClassification(t *testing.T) {
	ca, key := testProxyCA(t)
	cert := signedTestCertificate(t, ca, key, "custom-client", x509.ExtKeyUsageClientAuth)
	for name, tc := range map[string]struct {
		selector string
		path     string
		want     int
	}{
		"custom route": {selector: "Custom Reader", path: "/custom", want: http.StatusOK},
		"console selector cannot use custom route": {selector: accesscontrol.ConsoleProxy, path: "/custom", want: http.StatusForbidden},
		"console selector allowed route":           {selector: accesscontrol.ConsoleProxy, path: "/v1/deployments", want: http.StatusOK},
	} {
		t.Run(name, func(t *testing.T) {
			entry := clientPolicyEntry{AuthType: clientAuthMTLS, Subject: "custom-client",
				CARef: defaultClientPolicy().Clients[0].CARef, Access: []clientPolicyAccess{{PermissionSetName: tc.selector}}}
			if tc.selector == accesscontrol.ConsoleProxy {
				entry.Access[0].HTTPScopes = defaultClientPolicy().Clients[1].Access[0].HTTPScopes
			}
			policy := newStaticClientPolicy(t, clientPolicyDocument{Version: 1, Clients: []clientPolicyEntry{entry}}, clientCAConfigMap(ca))
			fixture, client := policyHandlerFixture(t, nil, policy)
			request := httptest.NewRequest(http.MethodGet, tc.path, nil)
			request.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert.Leaf}}
			response := httptest.NewRecorder()
			fixture.handler.ServeHTTP(response, request)
			assert.Equal(t, tc.want, response.Code)
			if tc.want == http.StatusOK {
				require.NotNil(t, client.lastRequest)
				require.Len(t, client.lastRequest.GetPermissionSets(), 1)
				assert.Equal(t, tc.selector, client.lastRequest.GetPermissionSets()[0].GetName())
				assert.True(t, client.lastRequest.GetClusterScopes()[0].GetFullClusterAccess())
			}
			centralcaps.Set([]centralsensor.CentralCapability{centralsensor.InternalTokenAPISupported})
			response = httptest.NewRecorder()
			fixture.handler.ServeHTTP(response, request)
			if tc.want == http.StatusForbidden {
				assert.Equal(t, http.StatusForbidden, response.Code)
			} else {
				assert.Equal(t, http.StatusNotImplemented, response.Code, "mTLS clients never use legacy console fallback")
			}
		})
	}
}

func TestDefaultMetricsClientStillMetricsOnly(t *testing.T) {
	ca, key := testProxyCA(t)
	cert := signedTestCertificate(t, ca, key, env.SecureMetricsClientCertCN.Setting(), x509.ExtKeyUsageClientAuth)
	metricsID := accesscontrol.OpenShiftPrometheusMetricsReaderID
	for selector, id := range map[string]string{
		"name":         "",
		"ID":           metricsID,
		"uppercase ID": strings.ToUpper(metricsID),
		"braced ID":    "{" + metricsID + "}",
		"compact ID":   strings.ReplaceAll(metricsID, "-", ""),
	} {
		t.Run(selector, func(t *testing.T) {
			doc := defaultClientPolicy()
			if id != "" {
				doc.Clients[0].Access[0].PermissionSetName = ""
				doc.Clients[0].Access[0].PermissionSetID = id
			}
			policy := newStaticClientPolicy(t, doc, clientCAConfigMap(ca))
			fixture, client := policyHandlerFixture(t, nil, policy)
			for name, tc := range map[string]struct {
				method, path string
				want         int
			}{
				"metrics":     {http.MethodGet, "/metrics", http.StatusOK},
				"POST":        {http.MethodPost, "/metrics", http.StatusForbidden},
				"other route": {http.MethodGet, "/custom", http.StatusForbidden},
			} {
				t.Run(name, func(t *testing.T) {
					request := httptest.NewRequest(tc.method, tc.path, nil)
					request.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert.Leaf}}
					response := httptest.NewRecorder()
					fixture.handler.ServeHTTP(response, request)
					assert.Equal(t, tc.want, response.Code)
					if tc.want == http.StatusOK && id != "" {
						require.NotNil(t, client.lastRequest)
						require.Len(t, client.lastRequest.GetPermissionSets(), 1)
						assert.Equal(t, metricsID, client.lastRequest.GetPermissionSets()[0].GetId())
					}
				})
			}
		})
	}
}
