package centralproxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/centralproxy/allowedpaths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func policyHandlerFixture(t *testing.T, authorizer *k8sAuthorizer, policy *ClientPolicy) (*proxyTestFixture, *fakeTokenServiceClient) {
	t.Helper()
	fixture := newProxyTestFixture(t, authorizer)
	centralcaps.Set([]centralsensor.CentralCapability{
		centralsensor.InternalTokenAPISupported, centralsensor.CentralProxyPathFiltering, centralsensor.CentralProxyPermissionSetTokens,
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
				AuthType: clientAuthKubernetesBearer, Subject: "test-user", PermissionSetName: "Custom Reader",
			}}})
			authorizer := newDenyingAuthorizer(t)
			if tc.sarAllowed {
				authorizer = newAllowingAuthorizer(t)
			}
			fixture, client := policyHandlerFixture(t, authorizer, policy)
			t.Setenv("ROX_OCP_CONSOLE_INTEGRATION", "false")
			allowedpaths.Set([]string{"/v1/metadata"})
			if !tc.capable {
				centralcaps.Set([]centralsensor.CentralCapability{centralsensor.InternalTokenAPISupported, centralsensor.CentralProxyPathFiltering})
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
				method         string
				path           string
				consoleEnabled bool
				legacy         bool
				want           int
			}{
				"GET":              {method: http.MethodGet, path: "/v1/metadata", consoleEnabled: true, want: http.StatusOK},
				"POST":             {method: http.MethodPost, path: "/v1/metadata", consoleEnabled: true, want: http.StatusOK},
				"HEAD":             {method: http.MethodHead, path: "/v1/metadata", consoleEnabled: true, want: http.StatusOK},
				"OPTIONS":          {method: http.MethodOptions, path: "/v1/metadata", consoleEnabled: true, want: http.StatusOK},
				"DELETE":           {method: http.MethodDelete, path: "/v1/metadata", consoleEnabled: true, want: http.StatusMethodNotAllowed},
				"unlisted route":   {method: http.MethodGet, path: "/custom", consoleEnabled: true, want: http.StatusForbidden},
				"console disabled": {method: http.MethodGet, path: "/v1/metadata", want: http.StatusForbidden},
				"legacy console":   {method: http.MethodGet, path: "/v1/metadata", consoleEnabled: true, legacy: true, want: http.StatusOK},
			} {
				t.Run(name, func(t *testing.T) {
					entry := clientPolicyEntry{AuthType: clientAuthKubernetesBearer, Subject: "test-user",
						PermissionSetID: id}
					if id == "" {
						entry.PermissionSetName = accesscontrol.ConsoleProxy
					}
					policy := newStaticClientPolicy(t, clientPolicyDocument{Version: 1, Clients: []clientPolicyEntry{entry}})
					fixture, client := policyHandlerFixture(t, newAllowingAuthorizer(t), policy)
					t.Setenv("ROX_OCP_CONSOLE_INTEGRATION", fmt.Sprint(tc.consoleEnabled))
					allowedpaths.Set([]string{"/v1/metadata"})
					if tc.legacy {
						centralcaps.Set([]centralsensor.CentralCapability{centralsensor.InternalTokenAPISupported, centralsensor.CentralProxyPathFiltering})
					}
					response := fixture.serveHTTP(t, tc.method, tc.path, map[string]string{"Authorization": "Bearer kube-token"})
					assert.Equal(t, tc.want, response.Code)
					assert.Equal(t, tc.want == http.StatusOK, fixture.proxyCalled)
					if tc.want == http.StatusOK {
						require.NotNil(t, client.lastRequest)
						assert.Empty(t, client.lastRequest.GetClusterScopes(), "console bootstrap must keep an empty scope")
						if tc.legacy {
							assert.Empty(t, client.lastRequest.GetPermissionSets())
							assert.Equal(t, tokenPermissions, client.lastRequest.GetPermissions())
						} else {
							assert.Empty(t, client.lastRequest.GetPermissions())
							if id != "" {
								require.Len(t, client.lastRequest.GetPermissionSets(), 1)
								assert.Equal(t, consoleID, client.lastRequest.GetPermissionSets()[0].GetId())
							}
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
		"console selector allowed route":           {selector: accesscontrol.ConsoleProxy, path: "/v1/metadata", want: http.StatusOK},
	} {
		t.Run(name, func(t *testing.T) {
			entry := clientPolicyEntry{AuthType: clientAuthMTLS, Subject: "custom-client",
				CARef: testClientCARef(t), PermissionSetName: tc.selector}
			policy := newStaticClientPolicy(t, clientPolicyDocument{Version: 1, Clients: []clientPolicyEntry{entry}}, clientCAConfigMap(ca))
			fixture, client := policyHandlerFixture(t, nil, policy)
			request := httptest.NewRequest(http.MethodPost, tc.path, nil)
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
			assert.Equal(t, http.StatusNotImplemented, response.Code, "mTLS clients never use legacy console fallback")
		})
	}
}
