package centralproxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/pkg/errors"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	authenticationv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"
)

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

	proxy := &httputil.ReverseProxy{
		Transport: transport,
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(baseURL)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			pkghttputil.WriteError(w,
				pkghttputil.Errorf(http.StatusInternalServerError, "failed to contact central: %v", err),
			)
		},
	}

	return &Handler{
		clusterIDGetter: &testClusterIDGetter{clusterID: "test-cluster-id"},
		authorizer:      authorizer,
		proxy:           proxy,
	}
}

// newTestHandlerWithTransportError creates a Handler for testing where the transport returns an error.
func newTestHandlerWithTransportError(t *testing.T, baseURL *url.URL, authorizer *k8sAuthorizer, transportErr error) *Handler {
	t.Helper()

	transport := &mockTokenTransport{
		err: transportErr,
	}

	proxy := &httputil.ReverseProxy{
		Transport: transport,
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(baseURL)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			// Match production error handling: return 503 for initialization errors
			if errors.Is(err, errServiceUnavailable) {
				pkghttputil.WriteError(w,
					pkghttputil.Errorf(http.StatusServiceUnavailable, "proxy temporarily unavailable: %v", err),
				)
				return
			}
			pkghttputil.WriteError(w,
				pkghttputil.Errorf(http.StatusInternalServerError, "failed to contact central: %v", err),
			)
		},
	}

	return &Handler{
		clusterIDGetter: &testClusterIDGetter{clusterID: "test-cluster-id"},
		authorizer:      authorizer,
		proxy:           proxy,
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
