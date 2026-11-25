package centralproxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"

	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/centralproxy/allowedpaths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
)

func accessPolicy(grants ...clientPolicyAccess) clientPolicyDocument {
	return clientPolicyDocument{Version: 1, Clients: []clientPolicyEntry{{
		AuthType: clientAuthKubernetesBearer, Subject: "test-user", Access: grants,
	}}}
}

func TestPerAccessLocalScopeSelection(t *testing.T) {
	grants := []clientPolicyAccess{
		{PermissionSetName: "Deployment Reader", HTTPScopes: tokens.HTTPRequestScopes{
			"/v1/": {"GET"}, "/v1/methods": {"GET"}, "/v1/private/": {},
		}},
		{PermissionSetName: "Image Reader", HTTPScopes: tokens.HTTPRequestScopes{
			"/v1/images": {"POST"}, "/v1/methods": {"POST"},
		}},
		{PermissionSetName: "Deny Routes", HTTPScopes: tokens.HTTPRequestScopes{"/v1/blocked/": nil}},
	}
	policy := newStaticClientPolicy(t, accessPolicy(grants...))
	for name, tc := range map[string]struct {
		method, path string
		allowed      bool
	}{
		"broad prefix":               {"GET", "/v1/deployments", true},
		"specific route":             {"POST", "/v1/images", true},
		"overlap GET":                {"GET", "/v1/methods", true},
		"overlap POST":               {"POST", "/v1/methods", true},
		"no broader method fallback": {"GET", "/v1/images", false},
		"encoded URL":                {"POST", "/v1/%69mages", false},
		"explicit denial":            {"GET", "/v1/private/", false},
		"denied subtree":             {"GET", "/v1/private/image", false},
		"deny only":                  {"GET", "/v1/blocked/", false},
		"unlisted path":              {"GET", "/other", false},
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, tc.path, nil)
			request.Header.Set("Authorization", "Bearer reviewed")
			grant, err := policy.Authorize(t.Context(), request, &authenticationv1.UserInfo{Username: "test-user"})
			if !tc.allowed {
				require.Error(t, err)
				assert.Nil(t, grant)
				return
			}
			require.NoError(t, err)
			protoassert.SlicesEqual(t, clientPolicyAccessToProto(grants), grant.PermissionSets)
		})
	}
}

func TestPerAccessConfigBoundsAndValidation(t *testing.T) {
	for name, tc := range map[string]struct {
		grants, paths, methods int
		valid                  bool
	}{
		"32 unrestricted grants": {grants: 32, valid: true},
		"33 unrestricted grants": {grants: 33},
		"32 paths":               {grants: 1, paths: 32, methods: 1, valid: true},
		"33 paths":               {grants: 1, paths: 33, methods: 1},
		"aggregate 32 methods":   {grants: 2, paths: 8, methods: 2, valid: true},
		"aggregate 36 methods":   {grants: 2, paths: 9, methods: 2},
		"no grants":              {},
	} {
		t.Run(name, func(t *testing.T) {
			var grants []clientPolicyAccess
			for i := range tc.grants {
				grant := clientPolicyAccess{PermissionSetName: fmt.Sprintf("Reader %d", i)}
				if tc.paths > 0 {
					grant.HTTPScopes = make(tokens.HTTPRequestScopes)
					for j := range tc.paths {
						grant.HTTPScopes[fmt.Sprintf("/route/%d", j)] = []string{"GET", "POST"}[:tc.methods]
					}
				}
				grants = append(grants, grant)
			}
			_, err := parseClientPolicy(policyConfigMap(t, accessPolicy(grants...)))
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
	for name, scopes := range map[string]tokens.HTTPRequestScopes{
		"empty map": {}, "lowercase method": {"/v1/": {"get"}}, "empty method": {"/v1/": {""}},
		"encoded path": {"/%61": {"GET"}}, "noncanonical path": {"/v1/../metrics": {"GET"}},
		"duplicate method": {"/v1/": {"GET", "GET"}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseClientPolicy(policyConfigMap(t, accessPolicy(clientPolicyAccess{PermissionSetName: "Reader", HTTPScopes: scopes})))
			require.Error(t, err)
		})
	}
	_, err := parseClientPolicy(policyConfigMap(t, accessPolicy(
		clientPolicyAccess{PermissionSetName: "Reader"}, clientPolicyAccess{PermissionSetName: "Reader"},
	)))
	require.Error(t, err)
	doc := defaultClientPolicy()
	doc.Clients[1].Access = append(doc.Clients[1].Access, clientPolicyAccess{PermissionSetName: "Custom"})
	_, err = parseClientPolicy(policyConfigMap(t, doc))
	require.Error(t, err, "a wildcard must never acquire a custom grant")
}

func TestMixedGrantsKeepCustomScopeAndCapabilityRequirements(t *testing.T) {
	for order, reverse := range map[string]bool{"console first": false, "console last": true} {
		t.Run(order, func(t *testing.T) {
			grants := []clientPolicyAccess{
				defaultClientPolicy().Clients[1].Access[0],
				{PermissionSetName: "Custom Reader"},
			}
			if reverse {
				slices.Reverse(grants)
			}
			for name, tc := range map[string]struct {
				namespace    string
				sar, capable bool
				want         int
			}{
				"console disabled custom works":    {namespace: "prod", sar: true, capable: true, want: http.StatusOK},
				"namespace required":               {sar: true, capable: true, want: http.StatusForbidden},
				"SAR required":                     {namespace: "prod", capable: true, want: http.StatusForbidden},
				"single selector cap insufficient": {namespace: "prod", sar: true, want: http.StatusNotImplemented},
			} {
				t.Run(name, func(t *testing.T) {
					authorizer := newDenyingAuthorizer(t)
					if tc.sar {
						authorizer = newAllowingAuthorizer(t)
					}
					fixture, client := policyHandlerFixture(t, authorizer, newStaticClientPolicy(t, accessPolicy(grants...)))
					t.Setenv("ROX_OCP_CONSOLE_INTEGRATION", "false")
					if !tc.capable {
						centralcaps.Set([]centralsensor.CentralCapability{centralsensor.InternalTokenAPISupported, centralsensor.CentralProxyPermissionSetTokens})
					}
					response := fixture.serveHTTP(t, http.MethodDelete, "/custom/route", map[string]string{
						"Authorization": "Bearer reviewed", stackroxNamespaceHeader: tc.namespace,
					})
					assert.Equal(t, tc.want, response.Code)
					if tc.want == http.StatusOK {
						// Only the unrestricted custom grant admits the /custom route.
						require.Len(t, client.lastRequest.GetPermissionSets(), 1)
						assert.Equal(t, "Custom Reader", client.lastRequest.GetPermissionSets()[0].GetName())
						assert.Empty(t, client.lastRequest.GetPermissions())
					} else {
						assert.Nil(t, client.lastRequest)
					}
				})
			}
		})
	}
}

func TestHTTPGrantCapabilityFallback(t *testing.T) {
	for name, tc := range map[string]struct {
		single bool
		path   string
		access []clientPolicyAccess
		want   int
	}{
		"safe console legacy permissions": {path: "/v1/deployments", access: defaultClientPolicy().Clients[1].Access, want: http.StatusOK},
		"safe console single selector":    {single: true, path: "/v1/deployments", access: defaultClientPolicy().Clients[1].Access, want: http.StatusOK},
		"authentication-only legacy":      {path: "/v1/metadata", access: defaultClientPolicy().Clients[1].Access, want: http.StatusOK},
		"unrestricted console rejected":   {single: true, path: "/v1/metadata", access: []clientPolicyAccess{{PermissionSetName: accesscontrol.K8sCentralProxyUser}}, want: http.StatusNotImplemented},
		"console extra route rejected": {single: true, path: "/v1/metadata", access: []clientPolicyAccess{{PermissionSetName: accesscontrol.K8sCentralProxyUser,
			HTTPScopes: tokens.HTTPRequestScopes{"/v1/metadata": {"GET"}, "/custom": {"GET"}}}}, want: http.StatusNotImplemented},
		"scoped custom rejected": {single: true, path: "/v1/metadata", access: []clientPolicyAccess{{PermissionSetName: "Custom Reader",
			HTTPScopes: tokens.HTTPRequestScopes{"/v1/metadata": {"GET"}}}}, want: http.StatusNotImplemented},
		"unscoped custom rejected": {single: true, path: "/v1/metadata", access: []clientPolicyAccess{{PermissionSetName: "Custom Reader"}}, want: http.StatusNotImplemented},
	} {
		t.Run(name, func(t *testing.T) {
			fixture, client := policyHandlerFixture(t, newAllowingAuthorizer(t), newStaticClientPolicy(t, accessPolicy(tc.access...)))
			caps := []centralsensor.CentralCapability{centralsensor.InternalTokenAPISupported, centralsensor.CentralProxyPathFiltering}
			if tc.single {
				caps = append(caps, centralsensor.CentralProxyPermissionSetTokens)
			}
			centralcaps.Set(caps)
			allowedpaths.Set([]string{"/v1/metadata", "/v1/deployments"})
			response := fixture.serveHTTP(t, http.MethodGet, tc.path, map[string]string{
				"Authorization": "Bearer reviewed", stackroxNamespaceHeader: "prod",
			})
			assert.Equal(t, tc.want, response.Code)
			if tc.want != http.StatusOK {
				assert.Nil(t, client.lastRequest)
				return
			}
			switch {
			case tc.single:
				require.Len(t, client.lastRequest.GetPermissionSets(), 1)
				assert.Empty(t, client.lastRequest.GetPermissions())
			case tc.path == "/v1/metadata":
				assert.Empty(t, client.lastRequest.GetPermissionSets(), "authentication-only routes carry no permission sets")
				assert.Empty(t, client.lastRequest.GetPermissions(), "authentication-only grants request no permissions on legacy Centrals")
			default:
				assert.Empty(t, client.lastRequest.GetPermissionSets())
				assert.Equal(t, tokenPermissions, client.lastRequest.GetPermissions())
			}
		})
	}
}

func TestPerAccessScopeChangesInvalidateCachedProfile(t *testing.T) {
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.CentralProxyPerAccessTokens})
	t.Cleanup(func() { centralcaps.Set(nil) })
	var calls atomic.Int32
	client := &fakeTokenServiceClient{response: &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "first"}, callCount: &calls}
	provider := newTestTokenProvider(client, "sensor-cluster")
	profile := consoleTestProfile("prod")
	first, err := provider.getTokenForProfile(t.Context(), profile)
	require.NoError(t, err)
	other := *profile
	other.permissionSets = []*centralv1.SelectedPermissionSet{profile.permissionSets[0].CloneVT()}
	other.permissionSets[0].HttpRequestScopes = other.permissionSets[0].GetHttpRequestScopes()[:1]
	client.response = &centralv1.GenerateTokenForPermissionsAndScopeResponse{Token: "second"}
	second, err := provider.getTokenForProfile(t.Context(), &other)
	require.NoError(t, err)
	assert.NotEqual(t, first, second)
	again, err := provider.getTokenForProfile(t.Context(), profile)
	require.NoError(t, err)
	assert.Equal(t, first, again)
	assert.EqualValues(t, 2, calls.Load())
}
