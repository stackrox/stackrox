package centralproxy

import (
	"testing"

	authenticationv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"
)

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
