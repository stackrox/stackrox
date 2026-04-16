package centralproxy

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/centralsensor"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/centralproxy/allowedpaths"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"
)

// newReverseProxyForTest creates a reverse proxy with the given transport and base URL.
// It uses the shared proxyErrorHandler from handler.go to ensure consistent error handling
// between production and test code.
func newReverseProxyForTest(baseURL *url.URL, transport http.RoundTripper) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Transport:    transport,
		Rewrite:      func(r *httputil.ProxyRequest) { r.SetURL(baseURL) },
		ErrorHandler: proxyErrorHandler,
	}
}

// testClusterIDGetter is a test implementation of clusterIDGetter.
type testClusterIDGetter struct {
	clusterID string
}

func (t *testClusterIDGetter) GetNoWait() string {
	return t.clusterID
}

// mockTokenTransport is a test RoundTripper that injects a static token or returns an error.
type mockTokenTransport struct {
	base  http.RoundTripper
	token string
	err   error
}

func (m *mockTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.base == nil {
		return nil, errors.New("mockTokenTransport: base transport is nil")
	}
	reqCopy := req.Clone(req.Context())
	reqCopy.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.token))
	return m.base.RoundTrip(reqCopy) //nolint:wrapcheck
}

// newTestHandler creates a Handler for testing with the given components.
func newTestHandler(t *testing.T, baseURL *url.URL, baseTransport http.RoundTripper, authorizer *k8sAuthorizer, token string) *Handler {
	t.Helper()

	transport := &mockTokenTransport{
		base:  baseTransport,
		token: token,
	}

	return &Handler{
		clusterIDGetter: &testClusterIDGetter{clusterID: "test-cluster-id"},
		authorizer:      authorizer,
		proxy:           newReverseProxyForTest(baseURL, transport),
	}
}

// newTestHandlerWithTransportError creates a Handler for testing where the transport returns an error.
func newTestHandlerWithTransportError(t *testing.T, baseURL *url.URL, authorizer *k8sAuthorizer, transportErr error) *Handler {
	t.Helper()

	transport := &mockTokenTransport{
		err: transportErr,
	}

	return &Handler{
		clusterIDGetter: &testClusterIDGetter{clusterID: "test-cluster-id"},
		authorizer:      authorizer,
		proxy:           newReverseProxyForTest(baseURL, transport),
	}
}

// newAllowingAuthorizer creates a k8sAuthorizer with a fake client that allows all authorization requests.
func newAllowingAuthorizer(t testing.TB) *k8sAuthorizer {
	t.Helper()
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

	// Mock SubjectAccessReview to allow all
	fakeClient.PrependReactor("create", "subjectaccessreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
		return true, &authv1.SubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: true,
			},
		}, nil
	})

	return newK8sAuthorizer(fakeClient)
}

// newDenyingAuthorizer creates a k8sAuthorizer with a fake client that denies all authorization requests.
// Authentication succeeds but authorization (SAR) fails.
func newDenyingAuthorizer(t testing.TB) *k8sAuthorizer {
	t.Helper()
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

	// Mock SubjectAccessReview to deny all
	fakeClient.PrependReactor("create", "subjectaccessreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
		return true, &authv1.SubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: false,
			},
		}, nil
	})

	return newK8sAuthorizer(fakeClient)
}

// newUnauthenticatedAuthorizer creates a k8sAuthorizer with a fake client that rejects all tokens.
// Use this to test authentication failures (token validation fails).
func newUnauthenticatedAuthorizer(t testing.TB) *k8sAuthorizer {
	t.Helper()
	fakeClient := fake.NewClientset()

	// Mock TokenReview to return unauthenticated (invalid token)
	fakeClient.PrependReactor("create", "tokenreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
		return true, &authenticationv1.TokenReview{
			Status: authenticationv1.TokenReviewStatus{
				Authenticated: false,
			},
		}, nil
	})

	return newK8sAuthorizer(fakeClient)
}

// errTransportError is a sentinel error for transport failures in tests.
var errTransportError = errors.New("transport error")

// setupCentralCapsForTest sets the Central capabilities required for the proxy to function.
// It also configures the allowed proxy paths to include the test paths used by existing tests.
// It registers a cleanup function that clears the capabilities and allowed paths.
func setupCentralCapsForTest(t *testing.T) {
	t.Helper()
	centralcaps.Set([]centralsensor.CentralCapability{
		centralsensor.InternalTokenAPISupported,
		centralsensor.CentralProxyPathFiltering,
	})
	allowedpaths.Set([]string{"/v1/", "/v2/", "/api/graphql", "/api/v1/"})
	t.Cleanup(func() {
		centralcaps.Set(nil)
		allowedpaths.Reset()
	})
}

// proxyTestFixture bundles the boilerplate shared by the majority of handler
// subtests: central caps, a mock transport that tracks whether it was called,
// a parsed base URL, a handler wired with the given authorizer and a static
// token, and centralReachable set to true.
type proxyTestFixture struct {
	handler     *Handler
	proxyCalled bool
}

// newProxyTestFixture creates a ready-to-use test fixture.
// It registers a cleanup function that resets central caps and allowed paths.
func newProxyTestFixture(t *testing.T, authorizer *k8sAuthorizer) *proxyTestFixture {
	t.Helper()
	setupCentralCapsForTest(t)

	f := &proxyTestFixture{}

	mockTransport := pkghttputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		f.proxyCalled = true
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	baseURL, err := url.Parse("https://central:443")
	require.NoError(t, err)

	f.handler = newTestHandler(t, baseURL, mockTransport, authorizer, "test-token")
	f.handler.centralReachable.Store(true)
	return f
}

// serveHTTP creates an HTTP request with the given method, path, and headers,
// calls ServeHTTP on the fixture's handler, and returns the response recorder.
func (f *proxyTestFixture) serveHTTP(t *testing.T, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	return w
}
