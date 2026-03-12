//go:build test_e2e

package tests

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	routeclient "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// retryTimeout is the maximum duration allowed for a network call to eventually succeed.
	retryTimeout = 60 * time.Second
	// retryInterval is the polling interval between successive retry attempts.
	retryInterval = 5 * time.Second

	// sensorProxyServiceName is the Kubernetes Service name for the Sensor proxy.
	sensorProxyServiceName = "sensor-proxy"
	// sensorProxyPort is the TCP port Sensor exposes for the OCP console plugin proxy.
	sensorProxyPort = 9444

	// deploymentsGraphQLQuery requests deployment IDs filtered to apps/v1 Deployments
	// so the count matches the Kubernetes API listing.
	deploymentsGraphQLQuery = `{"query":"{deployments(query:\"Deployment Type:Deployment\"){id}}"}`
)

// graphQLDeploymentsResponse is the response structure for the deployments GraphQL query.
// Only the id field is requested, so each element is an empty struct; only the count matters.
type graphQLDeploymentsResponse struct {
	Data struct {
		Deployments []struct{} `json:"deployments"`
	} `json:"data"`
}

// testNamespace is the namespace used for all test resources and scoping assertions.
// It can be overridden with the ACS_E2E_TEST_NAMESPACE environment variable.
var testNamespace = func() string {
	if ns := os.Getenv("ACS_E2E_TEST_NAMESPACE"); ns != "" {
		return ns
	}
	return "stackrox"
}()

// newInsecureHTTPClient creates an HTTP client that skips TLS verification.
// This is required when connecting to Sensor's proxy, which presents a Service CA signed cert.
func newInsecureHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 30 * time.Second,
	}
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

// setupProxyNetworkPolicy creates a temporary NetworkPolicy that allows the OpenShift
// Router (in openshift-ingress namespace) to reach the sensor-proxy port sensorProxyPort.
// This is required because the production NetworkPolicy restricts that port to
// openshift-console/console pods only; E2E tests access the proxy via an OCP Route,
// which routes through the openshift-ingress namespace.
func setupProxyNetworkPolicy(t *testing.T, ctx context.Context, k8sClient kubernetes.Interface) {
	t.Helper()
	policyName := fmt.Sprintf("acs-ocp-e2e-%s", uuid.NewV4())

	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      policyName,
			Namespace: testNamespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metaV1.LabelSelector{
				MatchLabels: map[string]string{"app": "sensor"},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metaV1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": "openshift-ingress",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: pointers.Pointer(corev1.ProtocolTCP),
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: sensorProxyPort},
						},
					},
				},
			},
		},
	}
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		_, err := k8sClient.NetworkingV1().NetworkPolicies(testNamespace).Create(ctx, policy, metaV1.CreateOptions{})
		if err != nil && !k8sErrors.IsAlreadyExists(err) {
			assert.NoError(c, err, "creating temporary E2E NetworkPolicy %s", policyName)
		}
	}, retryTimeout, retryInterval)
	t.Cleanup(func() {
		_ = k8sClient.NetworkingV1().NetworkPolicies(testNamespace).Delete(
			context.Background(), policyName, metaV1.DeleteOptions{})
	})
}

// setupSensorRoute creates a temporary OpenShift Route backed by the sensorProxyServiceName
// Service and waits until the OCP router assigns a hostname. It returns the hostname assigned
// by the router.
func setupSensorRoute(t *testing.T, ctx context.Context, restCfg *rest.Config) string {
	t.Helper()

	rc, err := routeclient.NewForConfig(restCfg)
	require.NoError(t, err, "creating OpenShift route client")

	routeName := fmt.Sprintf("acs-ocp-e2e-%s", uuid.NewV4())

	route := &routev1.Route{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      routeName,
			Namespace: testNamespace,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: sensorProxyServiceName,
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("https"),
			},
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationPassthrough,
			},
		},
	}

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		_, err := rc.RouteV1().Routes(testNamespace).Create(ctx, route, metaV1.CreateOptions{})
		if err != nil && !k8sErrors.IsAlreadyExists(err) {
			assert.NoError(c, err, "creating test Route %s in namespace %s", routeName, testNamespace)
		}
	}, retryTimeout, retryInterval)
	t.Cleanup(func() {
		_ = rc.RouteV1().Routes(testNamespace).Delete(context.Background(), routeName, metaV1.DeleteOptions{})
	})

	var routeHost string
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		r, err := rc.RouteV1().Routes(testNamespace).Get(ctx, routeName, metaV1.GetOptions{})
		require.NoError(c, err, "getting Route %s", routeName)
		require.NotEmpty(c, r.Status.Ingress,
			"Route %s has no ingress entries yet; waiting for OCP router to admit it", routeName)
		routeHost = r.Status.Ingress[0].Host
	}, retryTimeout, retryInterval)

	return routeHost
}

// setupSensorProxy creates a temporary OCP Route to the sensor-proxy Service and returns the
// proxy base URL.
func setupSensorProxy(t *testing.T, ctx context.Context, k8sClient kubernetes.Interface) string {
	t.Helper()
	restCfg := getConfig(t)

	// Allow OCP Router traffic to reach sensor port sensorProxyPort for the duration of this test.
	setupProxyNetworkPolicy(t, ctx, k8sClient)

	// Create a passthrough Route to sensor-proxy and wait for the router to assign a hostname.
	routeHost := setupSensorRoute(t, ctx, restCfg)

	return fmt.Sprintf("https://%s/proxy/central", routeHost)
}

// createServiceAccountAndToken creates a ServiceAccount in the given namespace, retries
// setupBindings (if non-nil) via EventuallyWithT to attach any required RBAC, then requests
// and returns a short-lived token. Callers must register RBAC resource cleanup with t.Cleanup
// before invoking this function, so that cleanup is registered exactly once regardless of
// how many retry attempts setupBindings requires.
func createServiceAccountAndToken(
	t *testing.T,
	ctx context.Context,
	k8sClient kubernetes.Interface,
	namespace, saName string,
	setupBindings func(c *assert.CollectT),
) string {
	t.Helper()

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		_, err := k8sClient.CoreV1().ServiceAccounts(namespace).Create(ctx, &corev1.ServiceAccount{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      saName,
				Namespace: namespace,
			},
		}, metaV1.CreateOptions{})
		if err != nil && !k8sErrors.IsAlreadyExists(err) {
			assert.NoError(c, err, "creating ServiceAccount %s in namespace %s", saName, namespace)
		}
	}, retryTimeout, retryInterval)
	t.Cleanup(func() {
		_ = k8sClient.CoreV1().ServiceAccounts(namespace).Delete(context.Background(), saName, metaV1.DeleteOptions{})
	})

	if setupBindings != nil {
		require.EventuallyWithT(t, setupBindings, retryTimeout, retryInterval)
	}

	var token string
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		tokenResp, err := k8sClient.CoreV1().ServiceAccounts(namespace).CreateToken(
			ctx, saName,
			&authv1.TokenRequest{
				Spec: authv1.TokenRequestSpec{
					ExpirationSeconds: pointers.Int64(600),
				},
			},
			metaV1.CreateOptions{},
		)
		require.NoError(c, err, "creating token for ServiceAccount %s in namespace %s", saName, namespace)
		token = tokenResp.Status.Token
	}, retryTimeout, retryInterval)
	return token
}

// setupNoScopeToken creates a ServiceAccount with no RBAC bindings in the given namespace and
// returns a short-lived token for it. Use this when the caller only needs a valid token that
// passes TokenReview without granting any list permissions.
func setupNoScopeToken(t *testing.T, ctx context.Context, k8sClient kubernetes.Interface, namespace, saName string) string {
	t.Helper()
	return createServiceAccountAndToken(t, ctx, k8sClient, namespace, saName, nil)
}

// setupNamespaceScopeToken creates a ServiceAccount named saName with view permissions in the
// given namespace and returns a short-lived token for it.
func setupNamespaceScopeToken(t *testing.T, ctx context.Context, k8sClient kubernetes.Interface, namespace, saName string) string {
	t.Helper()
	rbName := saName + "-view"
	t.Cleanup(func() {
		_ = k8sClient.RbacV1().RoleBindings(namespace).Delete(context.Background(), rbName, metaV1.DeleteOptions{})
	})
	return createServiceAccountAndToken(t, ctx, k8sClient, namespace, saName, func(c *assert.CollectT) {
		_, err := k8sClient.RbacV1().RoleBindings(namespace).Create(ctx, &rbacv1.RoleBinding{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      rbName,
				Namespace: namespace,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "view",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      saName,
					Namespace: namespace,
				},
			},
		}, metaV1.CreateOptions{})
		if err != nil && !k8sErrors.IsAlreadyExists(err) {
			assert.NoError(c, err, "creating RoleBinding %s for ServiceAccount %s in namespace %s", rbName, saName, namespace)
		}
	})
}

// setupClusterScopeToken creates a ServiceAccount named saName with cluster-wide view permissions
// via a ClusterRoleBinding and returns a short-lived token for it.
func setupClusterScopeToken(t *testing.T, ctx context.Context, k8sClient kubernetes.Interface, namespace, saName string) string {
	t.Helper()
	clusterRBName := saName + "-view-cluster"
	t.Cleanup(func() {
		_ = k8sClient.RbacV1().ClusterRoleBindings().Delete(
			context.Background(), clusterRBName, metaV1.DeleteOptions{})
	})
	return createServiceAccountAndToken(t, ctx, k8sClient, namespace, saName, func(c *assert.CollectT) {
		_, err := k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, &rbacv1.ClusterRoleBinding{
			ObjectMeta: metaV1.ObjectMeta{Name: clusterRBName},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "view",
			},
			Subjects: []rbacv1.Subject{{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: namespace,
			}},
		}, metaV1.CreateOptions{})
		if err != nil && !k8sErrors.IsAlreadyExists(err) {
			assert.NoError(c, err, "creating ClusterRoleBinding %s", clusterRBName)
		}
	})
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
	s.proxyBaseURL = setupSensorProxy(s.T(), s.ctx, s.k8sClient)

	// Three dedicated SAs for namespace-scoping tests, one per scenario, so each subtest
	// exercises an independent principal with exactly the permissions it requires.
	s.noScopeToken = setupNoScopeToken(s.T(), s.ctx, s.k8sClient, testNamespace, "acs-plugin-test-noscope-sa")
	s.nsScopeToken = setupNamespaceScopeToken(s.T(), s.ctx, s.k8sClient, testNamespace, "acs-plugin-test-ns-sa")
	s.clusterScopeToken = setupClusterScopeToken(s.T(), s.ctx, s.k8sClient, testNamespace, "acs-plugin-test-cluster-sa")
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
	s.T().Run("NoAuthorizationHeader", func(t *testing.T) {
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			resp := doHTTPRequest(c, s.ctx, s.client, http.MethodGet, s.proxyBaseURL+"/v1/metadata", nil, nil)
			defer utils.IgnoreError(resp.Body.Close)
			assert.Equal(c, http.StatusUnauthorized, resp.StatusCode,
				"proxy should return 401 when no Authorization header is provided")
		}, retryTimeout, retryInterval)
	})

	s.T().Run("InvalidBearerToken", func(t *testing.T) {
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			headers := map[string]string{"Authorization": "Bearer this-token-will-fail-tokenreview"}
			resp := doHTTPRequest(c, s.ctx, s.client, http.MethodGet, s.proxyBaseURL+"/v1/metadata", headers, nil)
			defer utils.IgnoreError(resp.Body.Close)
			assert.Equal(c, http.StatusUnauthorized, resp.StatusCode,
				"proxy should return 401 when the bearer token fails Kubernetes TokenReview")
		}, retryTimeout, retryInterval)
	})

	s.T().Run("MalformedOrUnsupportedAuthorizationHeader", func(t *testing.T) {
		t.Run("UnsupportedScheme", func(t *testing.T) {
			require.EventuallyWithT(t, func(c *assert.CollectT) {
				headers := map[string]string{"Authorization": "Token abc"}
				resp := doHTTPRequest(c, s.ctx, s.client, http.MethodGet, s.proxyBaseURL+"/v1/metadata", headers, nil)
				defer utils.IgnoreError(resp.Body.Close)
				assert.Equal(c, http.StatusUnauthorized, resp.StatusCode,
					"proxy should return 401 for an unsupported authorization scheme")
			}, retryTimeout, retryInterval)
		})
		t.Run("BearerWithoutToken", func(t *testing.T) {
			require.EventuallyWithT(t, func(c *assert.CollectT) {
				headers := map[string]string{"Authorization": "Bearer"}
				resp := doHTTPRequest(c, s.ctx, s.client, http.MethodGet, s.proxyBaseURL+"/v1/metadata", headers, nil)
				defer utils.IgnoreError(resp.Body.Close)
				assert.Equal(c, http.StatusUnauthorized, resp.StatusCode,
					"proxy should return 401 when Bearer scheme has no token")
			}, retryTimeout, retryInterval)
		})
	})
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
		// Filter to apps/v1/Deployment only so the count matches the K8s API listing.
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
			assert.Len(c, result.Data.Deployments, len(k8sList.Items),
				"proxy should return %d deployments for namespace %s", len(k8sList.Items), testNamespace)
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
		// Filter to apps/v1/Deployment only so the count matches the K8s API listing.
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
			assert.Len(c, result.Data.Deployments, len(k8sList.Items),
				"proxy should return %d deployments cluster-wide", len(k8sList.Items))
		}, retryTimeout, retryInterval)
	})
}
