//go:build test_e2e

package tests

import (
	"bytes"
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	routeclient "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	authv1 "k8s.io/api/authentication/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//go:embed testdata/ocp_plugin_resources.yaml
var ocpPluginResourcesYAML []byte

const (
	// retryTimeout is the maximum duration allowed for a network call to eventually succeed.
	retryTimeout = 60 * time.Second
	// retryInterval is the polling interval between successive retry attempts.
	retryInterval = 5 * time.Second

	// deploymentsGraphQLQuery requests deployment name and namespace filtered to apps/v1
	// Deployments so the results can be compared against the Kubernetes API listing.
	deploymentsGraphQLQuery = `{"query":"{deployments(query:\"Deployment Type:Deployment\"){name namespace}}"}`
)

// graphQLDeploymentsResponse is the response structure for the deployments GraphQL query.
type graphQLDeploymentsResponse struct {
	Data struct {
		Deployments []struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"deployments"`
	} `json:"data"`
}

// testNamespace is the namespace used for all test resources and scoping assertions.
const testNamespace = "stackrox"

// newInsecureHTTPClient creates a retryable HTTP client that skips TLS verification.
// TLS verification is skipped because Sensor's proxy presents a Service CA signed cert.
// Logger is silenced to avoid polluting test output with retry attempt messages.
func newInsecureHTTPClient() *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	retryClient.HTTPClient.Timeout = 30 * time.Second
	retryClient.Logger = nil
	return retryClient.StandardClient()
}

// doHTTPRequest creates and executes an HTTP request, returning the response.
// Pass nil for headers when no custom headers are needed.
// t may be a *testing.T or *assert.CollectT to support use inside require.EventuallyWithT.
func doHTTPRequest(t require.TestingT, ctx context.Context, client *http.Client, method, rawURL string, headers map[string]string, body io.Reader) *http.Response {
	if h, ok := t.(interface{ Helper() }); ok {
		h.Helper()
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	require.NoError(t, err, "creating HTTP request for %s %s", method, rawURL)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	require.NoError(t, err, "executing HTTP request to %s %s", method, rawURL)
	return resp
}

// applyYAMLResources applies the embedded prerequisite manifests via `oc apply` and
// registers a cleanup that removes them with `oc delete --ignore-not-found`.
func applyYAMLResources(t *testing.T, ctx context.Context) {
	t.Helper()
	ocRun := func(ctx context.Context, args ...string) {
		cmd := exec.CommandContext(ctx, "oc", args...)
		cmd.Stdin = bytes.NewReader(ocpPluginResourcesYAML)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "oc %s: %s", strings.Join(args, " "), string(out))
	}
	ocRun(ctx, "apply", "--wait", "-n", testNamespace, "-f", "-")
	t.Cleanup(func() {
		ocRun(context.Background(), "delete", "--wait", "--ignore-not-found", "-n", testNamespace, "-f", "-")
	})
}

// waitForRouteHost polls until the OCP router assigns a hostname to the named Route.
func waitForRouteHost(t *testing.T, ctx context.Context, rc routeclient.Interface, routeName string) string {
	t.Helper()
	var host string
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		r, err := rc.RouteV1().Routes(testNamespace).Get(ctx, routeName, metaV1.GetOptions{})
		require.NoError(c, err, "getting Route %s", routeName)
		require.NotEmpty(c, r.Status.Ingress,
			"Route %s has no ingress entries yet; waiting for OCP router to admit it", routeName)
		host = r.Status.Ingress[0].Host
	}, retryTimeout, retryInterval)
	return host
}

// requestToken creates and returns a short-lived token for the named ServiceAccount.
func requestToken(t *testing.T, ctx context.Context, k8sClient kubernetes.Interface, namespace, saName string) string {
	t.Helper()
	var token string
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		resp, err := k8sClient.CoreV1().ServiceAccounts(namespace).CreateToken(
			ctx, saName,
			&authv1.TokenRequest{Spec: authv1.TokenRequestSpec{ExpirationSeconds: pointers.Int64(600)}},
			metaV1.CreateOptions{},
		)
		require.NoError(c, err, "creating token for ServiceAccount %s/%s", namespace, saName)
		token = resp.Status.Token
	}, retryTimeout, retryInterval)
	return token
}

// OCPPluginSuite shares one sensor-proxy setup and one set of test ServiceAccounts across
// all OCP plugin tests.
type OCPPluginSuite struct {
	suite.Suite
	ctx               context.Context
	client            *http.Client
	k8sClient         kubernetes.Interface
	proxyBaseURL      string
	noScopeToken      string
	nsScopeToken      string
	clusterScopeToken string
}

func TestOCPPlugin(t *testing.T) {
	suite.Run(t, new(OCPPluginSuite))
}

func (s *OCPPluginSuite) SetupSuite() {
	if !isOpenshift() {
		s.T().Skip("OCP console plugin tests require an OpenShift cluster (ORCHESTRATOR_FLAVOR=openshift).")
	}
	s.ctx = context.Background()
	s.client = newInsecureHTTPClient()
	s.k8sClient = createK8sClient(s.T())

	rc, err := routeclient.NewForConfig(getConfig(s.T()))
	require.NoError(s.T(), err, "creating OpenShift route client")

	// Phase 1: apply all prerequisite resources defined in testdata/ocp_plugin_resources.yaml.
	applyYAMLResources(s.T(), s.ctx)

	// Phase 2: extract values that require reading back from the API.
	routeHost := waitForRouteHost(s.T(), s.ctx, rc, "acs-ocp-e2e-proxy-route")
	s.proxyBaseURL = fmt.Sprintf("https://%s/proxy/central", routeHost)

	s.noScopeToken = requestToken(s.T(), s.ctx, s.k8sClient, testNamespace, "acs-plugin-test-noscope-sa")
	s.nsScopeToken = requestToken(s.T(), s.ctx, s.k8sClient, testNamespace, "acs-plugin-test-ns-sa")
	s.clusterScopeToken = requestToken(s.T(), s.ctx, s.k8sClient, testNamespace, "acs-plugin-test-cluster-sa")
}

// TestProxyPathEnforcement verifies the Sensor proxy's allowlist enforcement:
// allowed paths are forwarded and return 200, disallowed paths are blocked with 403.
// A 200 on allowed paths also proves the proxy substituted a valid Central internal token
// (a raw K8s token would be rejected by Central).
func (s *OCPPluginSuite) TestProxyPathEnforcement() {
	authHeader := map[string]string{"Authorization": "Bearer " + s.noScopeToken}

	testCases := []struct {
		name   string
		method string
		path   string
		// getBody returns a fresh request body for each retry attempt.
		// nil means no body (e.g. GET requests).
		getBody    func() io.Reader
		headers    map[string]string
		wantStatus int
	}{
		// Allowed paths — expect 200.
		{name: "Metadata", method: http.MethodGet, path: "/v1/metadata", headers: authHeader, wantStatus: http.StatusOK},
		{name: "FeatureFlags", method: http.MethodGet, path: "/v1/featureflags", headers: authHeader, wantStatus: http.StatusOK},
		{name: "PublicConfig", method: http.MethodGet, path: "/v1/config/public", headers: authHeader, wantStatus: http.StatusOK},
		{name: "MyPermissions", method: http.MethodGet, path: "/v1/mypermissions", headers: authHeader, wantStatus: http.StatusOK},
		{name: "Deployments", method: http.MethodGet, path: "/v1/deployments", headers: authHeader, wantStatus: http.StatusOK},
		{
			name:    "GraphQL",
			method:  http.MethodPost,
			path:    "/api/graphql",
			getBody: func() io.Reader { return strings.NewReader(`{"query":"{metadata{version}}"}`) },
			headers: map[string]string{
				"Authorization": "Bearer " + s.noScopeToken,
				"Content-Type":  "application/json",
			},
			wantStatus: http.StatusOK,
		},
		{name: "StaticPlugin", method: http.MethodGet, path: "/static/ocp-plugin/plugin-manifest.json", headers: authHeader, wantStatus: http.StatusOK},
		// Disallowed paths — the proxy must block these before reaching Central.
		{name: "Alerts", method: http.MethodGet, path: "/v1/alerts", headers: authHeader, wantStatus: http.StatusForbidden},
		{name: "Policies", method: http.MethodGet, path: "/v1/policies", headers: authHeader, wantStatus: http.StatusForbidden},
		{name: "NetworkPolicies", method: http.MethodGet, path: "/v1/networkpolicies", headers: authHeader, wantStatus: http.StatusForbidden},
		{name: "AlertsPOST", method: http.MethodPost, path: "/v1/alerts", headers: authHeader, wantStatus: http.StatusForbidden},
		{name: "ArbitraryPath", method: http.MethodGet, path: "/v1/foobar", headers: authHeader, wantStatus: http.StatusForbidden},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			require.EventuallyWithT(t, func(c *assert.CollectT) {
				var body io.Reader
				if tc.getBody != nil {
					body = tc.getBody()
				}
				resp := doHTTPRequest(c, s.ctx, s.client, tc.method, s.proxyBaseURL+tc.path, tc.headers, body)
				defer utils.IgnoreError(resp.Body.Close)
				assert.Equal(c, tc.wantStatus, resp.StatusCode, "unexpected status for path %q", tc.path)
			}, retryTimeout, retryInterval)
		})
	}
}

// TestPluginManifest verifies that the plugin manifest served at the static path is valid JSON
// and contains the required "name" and "version" fields.
func (s *OCPPluginSuite) TestPluginManifest() {
	authHeader := map[string]string{"Authorization": "Bearer " + s.noScopeToken}
	require.EventuallyWithT(s.T(), func(c *assert.CollectT) {
		resp := doHTTPRequest(c, s.ctx, s.client, http.MethodGet,
			s.proxyBaseURL+"/static/ocp-plugin/plugin-manifest.json", authHeader, nil)
		defer utils.IgnoreError(resp.Body.Close)
		require.Equal(c, http.StatusOK, resp.StatusCode, "plugin manifest endpoint must return 200")
		assert.Contains(c, resp.Header.Get("Content-Type"), "application/json",
			"plugin manifest should be served as application/json")

		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(c, err, "reading plugin manifest response body")
		var manifest map[string]interface{}
		require.NoError(c, json.Unmarshal(bodyBytes, &manifest), "unmarshaling plugin manifest JSON")
		assert.Contains(c, manifest, "name", "plugin manifest must contain a 'name' field")
		assert.Contains(c, manifest, "version", "plugin manifest must contain a 'version' field")
	}, retryTimeout, retryInterval)
}

// TestProxyRequiresAuthentication verifies that the Sensor proxy returns 401 Unauthorized
// when requests arrive without a bearer token or with an invalid one.
func (s *OCPPluginSuite) TestProxyRequiresAuthentication() {
	testCases := []struct {
		name    string
		headers map[string]string
	}{
		{name: "NoAuthorizationHeader", headers: nil},
		{name: "InvalidBearerToken", headers: map[string]string{"Authorization": "Bearer this-token-will-fail-tokenreview"}},
		{name: "UnsupportedScheme", headers: map[string]string{"Authorization": "Token abc"}},
		{name: "BearerWithoutToken", headers: map[string]string{"Authorization": "Bearer"}},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			require.EventuallyWithT(t, func(c *assert.CollectT) {
				resp := doHTTPRequest(c, s.ctx, s.client, http.MethodGet, s.proxyBaseURL+"/v1/metadata", tc.headers, nil)
				defer utils.IgnoreError(resp.Body.Close)
				assert.Equal(c, http.StatusUnauthorized, resp.StatusCode,
					"proxy should return 401 for auth case %q", tc.name)
			}, retryTimeout, retryInterval)
		})
	}
}

// TestProxyNamespaceScoping verifies the ACS-AUTH-NAMESPACE-SCOPE header mechanism.
// When this header is set, Sensor performs a Kubernetes SubjectAccessReview before forwarding
// the request, restricting access to users with permissions in the specified namespace.
func (s *OCPPluginSuite) TestProxyNamespaceScoping() {
	s.T().Run("NoScope", func(t *testing.T) {
		// Without ACS-AUTH-NAMESPACE-SCOPE, no SubjectAccessReview is triggered and the proxy
		// returns zero deployments.
		headers := map[string]string{
			"Authorization": "Bearer " + s.noScopeToken,
			"Content-Type":  "application/json",
		}
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			body := strings.NewReader(deploymentsGraphQLQuery)
			resp := doHTTPRequest(c, s.ctx, s.client, http.MethodPost, s.proxyBaseURL+"/api/graphql", headers, body)
			defer utils.IgnoreError(resp.Body.Close)
			require.Equal(c, http.StatusOK, resp.StatusCode,
				"proxy should forward request without SAR when ACS-AUTH-NAMESPACE-SCOPE header is absent")
			var result graphQLDeploymentsResponse
			require.NoError(c, json.NewDecoder(resp.Body).Decode(&result), "decoding GraphQL response")
			assert.Empty(c, result.Data.Deployments,
				"proxy should return no deployments when ACS-AUTH-NAMESPACE-SCOPE header is absent")
		}, retryTimeout, retryInterval)
	})

	s.T().Run("NamespacedScope", func(t *testing.T) {
		// With ACS-AUTH-NAMESPACE-SCOPE set to a namespace where the SA has view permissions,
		// the SubjectAccessReview should pass and the proxy should return the deployments in
		// that namespace.
		// Filter to apps/v1/Deployment only so the names match the K8s API listing.
		headers := map[string]string{
			"Authorization":            "Bearer " + s.nsScopeToken,
			"ACS-AUTH-NAMESPACE-SCOPE": testNamespace,
			"Content-Type":             "application/json",
		}
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			k8sList, err := s.k8sClient.AppsV1().Deployments(testNamespace).List(s.ctx, metaV1.ListOptions{})
			require.NoError(c, err, "listing Kubernetes deployments in namespace %s", testNamespace)

			body := strings.NewReader(deploymentsGraphQLQuery)
			resp := doHTTPRequest(c, s.ctx, s.client, http.MethodPost, s.proxyBaseURL+"/api/graphql", headers, body)
			defer utils.IgnoreError(resp.Body.Close)
			require.Equal(c, http.StatusOK, resp.StatusCode,
				"proxy should allow request when SA has view permissions in the scoped namespace %s", testNamespace)
			var result graphQLDeploymentsResponse
			require.NoError(c, json.NewDecoder(resp.Body).Decode(&result), "decoding GraphQL response")

			graphQLNames := make([]string, len(result.Data.Deployments))
			for i, d := range result.Data.Deployments {
				graphQLNames[i] = d.Name
			}
			k8sNames := make([]string, len(k8sList.Items))
			for i, d := range k8sList.Items {
				k8sNames[i] = d.Name
			}
			assert.ElementsMatch(c, k8sNames, graphQLNames,
				"proxy should return the same deployments as Kubernetes for namespace %s", testNamespace)
		}, retryTimeout, retryInterval)
	})

	s.T().Run("NamespaceScopeDenied", func(t *testing.T) {
		// The noScopeToken SA has no RBAC bindings, so the SubjectAccessReview for
		// testNamespace is denied and the proxy should return 403 Forbidden.
		// This exercises the SAR failure path (authN passes, authZ fails).
		deniedHeaders := map[string]string{
			"Authorization":            "Bearer " + s.noScopeToken,
			"Content-Type":             "application/json",
			"ACS-AUTH-NAMESPACE-SCOPE": testNamespace,
		}
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			body := strings.NewReader(deploymentsGraphQLQuery)
			resp := doHTTPRequest(c, s.ctx, s.client, http.MethodPost, s.proxyBaseURL+"/api/graphql", deniedHeaders, body)
			defer utils.IgnoreError(resp.Body.Close)
			require.Equal(c, http.StatusForbidden, resp.StatusCode,
				"proxy should return 403 when SAR denies namespace-scoped access for a token with no RBAC bindings")
			bodyBytes, err := io.ReadAll(resp.Body)
			assert.NoError(c, err, "reading forbidden response body")
			assert.NotContains(c, string(bodyBytes), `"deployments"`,
				"denied namespace-scoped request should not return deployments data")
		}, retryTimeout, retryInterval)
	})

	s.T().Run("ClusterWideScope", func(t *testing.T) {
		// The wildcard scope (*) triggers a SAR for cluster-wide list permissions.
		// The SA has a ClusterRoleBinding so the SAR passes and the proxy returns 200.
		// Filter to apps/v1/Deployment only so the names match the K8s API listing.
		headers := map[string]string{
			"Authorization":            "Bearer " + s.clusterScopeToken,
			"ACS-AUTH-NAMESPACE-SCOPE": "*",
			"Content-Type":             "application/json",
		}
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			k8sList, err := s.k8sClient.AppsV1().Deployments("").List(s.ctx, metaV1.ListOptions{})
			require.NoError(c, err, "listing Kubernetes deployments cluster-wide")

			body := strings.NewReader(deploymentsGraphQLQuery)
			resp := doHTTPRequest(c, s.ctx, s.client, http.MethodPost, s.proxyBaseURL+"/api/graphql", headers, body)
			defer utils.IgnoreError(resp.Body.Close)
			require.Equal(c, http.StatusOK, resp.StatusCode,
				"proxy should allow cluster-wide request when SA has cluster-wide view permissions")
			var result graphQLDeploymentsResponse
			require.NoError(c, json.NewDecoder(resp.Body).Decode(&result), "decoding GraphQL response")

			// Use namespace/name pairs to uniquely identify deployments across namespaces.
			graphQLPairs := make([]string, len(result.Data.Deployments))
			for i, d := range result.Data.Deployments {
				graphQLPairs[i] = d.Namespace + "/" + d.Name
			}
			k8sPairs := make([]string, len(k8sList.Items))
			for i, d := range k8sList.Items {
				k8sPairs[i] = d.Namespace + "/" + d.Name
			}
			assert.ElementsMatch(c, k8sPairs, graphQLPairs,
				"proxy should return the same deployments as Kubernetes cluster-wide")
		}, retryTimeout, retryInterval)
	})
}
