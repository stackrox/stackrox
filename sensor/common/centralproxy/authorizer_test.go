package centralproxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/pkg/errors"
	pkghttputil "github.com/stackrox/rox/pkg/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"
)

func TestK8sAuthorizer_MissingToken(t *testing.T) {
	fakeClient := fake.NewClientset()
	authorizer := newK8sAuthorizer(fakeClient)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No Authorization header

	_, err := authorizer.authenticate(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing or invalid bearer token")
}

func TestK8sAuthorizer_InvalidToken(t *testing.T) {
	fakeClient := fake.NewClientset()
	authorizer := newK8sAuthorizer(fakeClient)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz") // Not Bearer token

	_, err := authorizer.authenticate(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing or invalid bearer token")
}

func TestK8sAuthorizer_TokenAuthenticationFailed(t *testing.T) {
	fakeClient := fake.NewClientset()

	// Mock TokenReview to return unauthenticated
	fakeClient.PrependReactor("create", "tokenreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
		return true, &authenticationv1.TokenReview{
			Status: authenticationv1.TokenReviewStatus{
				Authenticated: false,
			},
		}, nil
	})

	authorizer := newK8sAuthorizer(fakeClient)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	_, err := authorizer.authenticate(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token authentication failed")
}

func TestK8sAuthorizer_TokenReviewAPIError(t *testing.T) {
	fakeClient := fake.NewClientset()

	// Mock TokenReview to return API error
	fakeClient.PrependReactor("create", "tokenreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("API server unavailable")
	})

	authorizer := newK8sAuthorizer(fakeClient)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	_, err := authorizer.authenticate(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "performing token review")
	assert.Contains(t, err.Error(), "API server unavailable")
}

func TestK8sAuthorizer_TokenReviewStatusError(t *testing.T) {
	fakeClient := fake.NewClientset()

	// Mock TokenReview to return status error
	fakeClient.PrependReactor("create", "tokenreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
		return true, &authenticationv1.TokenReview{
			Status: authenticationv1.TokenReviewStatus{
				Authenticated: false,
				Error:         "token has expired",
			},
		}, nil
	})

	authorizer := newK8sAuthorizer(fakeClient)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	_, err := authorizer.authenticate(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token validation error")
	assert.Contains(t, err.Error(), "token has expired")
}

func TestK8sAuthorizer_AllPermissionsGranted_Namespace(t *testing.T) {
	fakeClient := fake.NewClientset()

	// Mock SubjectAccessReview - allow all
	fakeClient.PrependReactor("create", "subjectaccessreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
		return true, &authv1.SubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: true,
			},
		}, nil
	})

	authorizer := newK8sAuthorizer(fakeClient)

	userInfo := &authenticationv1.UserInfo{
		Username: "test-user",
		Groups:   []string{"test-group"},
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(stackroxNamespaceHeader, "test-namespace")

	err := authorizer.authorize(context.Background(), userInfo, req)

	assert.NoError(t, err)
}

func TestK8sAuthorizer_AllPermissionsGranted_ClusterWide(t *testing.T) {
	fakeClient := fake.NewClientset()

	// Mock SubjectAccessReview - allow all
	fakeClient.PrependReactor("create", "subjectaccessreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
		return true, &authv1.SubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: true,
			},
		}, nil
	})

	authorizer := newK8sAuthorizer(fakeClient)

	userInfo := &authenticationv1.UserInfo{
		Username: "cluster-admin",
		Groups:   []string{"system:masters"},
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Set cluster-wide scope header to trigger cluster-wide authorization check
	req.Header.Set(stackroxNamespaceHeader, FullClusterAccessScope)

	err := authorizer.authorize(context.Background(), userInfo, req)

	assert.NoError(t, err)
}

func TestK8sAuthorizer_MissingPermission_Namespace(t *testing.T) {
	fakeClient := fake.NewClientset()

	// Mock SubjectAccessReview - deny "list"
	fakeClient.PrependReactor("create", "subjectaccessreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8sTesting.CreateAction)
		sar := createAction.GetObject().(*authv1.SubjectAccessReview)

		// Allow "get", deny "list" and "watch"
		allowed := sar.Spec.ResourceAttributes.Verb == "get"

		return true, &authv1.SubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: allowed,
			},
		}, nil
	})

	authorizer := newK8sAuthorizer(fakeClient)

	userInfo := &authenticationv1.UserInfo{
		Username: "limited-user",
		Groups:   []string{"developers"},
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(stackroxNamespaceHeader, "my-namespace")

	err := authorizer.authorize(context.Background(), userInfo, req)

	assert.Error(t, err)
	// With parallel execution, any resource could fail first - check for the general pattern.
	assert.Contains(t, err.Error(), `user "limited-user" lacks LIST permission for resource`)
}

func TestK8sAuthorizer_MissingPermission_ClusterWide(t *testing.T) {
	fakeClient := fake.NewClientset()

	// Mock SubjectAccessReview - allow "get", deny "list"
	fakeClient.PrependReactor("create", "subjectaccessreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8sTesting.CreateAction)
		sar := createAction.GetObject().(*authv1.SubjectAccessReview)

		// Allow "get", deny "list"
		allowed := sar.Spec.ResourceAttributes.Verb == "get"

		return true, &authv1.SubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: allowed,
			},
		}, nil
	})

	authorizer := newK8sAuthorizer(fakeClient)

	userInfo := &authenticationv1.UserInfo{
		Username: "namespace-admin",
		Groups:   []string{"admins"},
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Set cluster-wide scope header to trigger cluster-wide authorization check
	req.Header.Set(stackroxNamespaceHeader, FullClusterAccessScope)

	err := authorizer.authorize(context.Background(), userInfo, req)

	assert.Error(t, err)
	// With parallel execution, any resource could fail first - check for the general pattern.
	assert.Contains(t, err.Error(), `user "namespace-admin" lacks cluster-wide LIST permission for resource`)
}

func TestK8sAuthorizer_SubjectAccessReviewError(t *testing.T) {
	fakeClient := fake.NewClientset()

	// Mock SubjectAccessReview to return error
	fakeClient.PrependReactor("create", "subjectaccessreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("API server unavailable")
	})

	authorizer := newK8sAuthorizer(fakeClient)

	userInfo := &authenticationv1.UserInfo{
		Username: "test-user",
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(stackroxNamespaceHeader, "test-namespace")

	err := authorizer.authorize(context.Background(), userInfo, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API server unavailable")
}

func TestK8sAuthorizer_SubjectAccessReviewEvaluationError(t *testing.T) {
	fakeClient := fake.NewClientset()

	// Mock SubjectAccessReview: Allowed = true but with an EvaluationError.
	// performSubjectAccessReview should return a 500 error instead of treating it as denial.
	fakeClient.PrependReactor("create", "subjectaccessreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
		return true, &authv1.SubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed:         true,
				EvaluationError: "some evaluation error from API server",
			},
		}, nil
	})

	authorizer := newK8sAuthorizer(fakeClient)

	userInfo := &authenticationv1.UserInfo{
		Username: "test-user",
		Groups:   []string{"test-group"},
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(stackroxNamespaceHeader, "test-namespace")

	err := authorizer.authorize(context.Background(), userInfo, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authorization evaluation error")
	assert.Contains(t, err.Error(), "some evaluation error from API server")

	// Verify it's a 500 Internal Server Error
	httpErr, ok := err.(pkghttputil.HTTPError)
	assert.True(t, ok, "error should be an HTTPError")
	assert.Equal(t, http.StatusInternalServerError, httpErr.HTTPStatusCode())
}

func TestK8sAuthorizer_CachingBehavior(t *testing.T) {
	fakeClient := fake.NewClientset()

	sarCallCount := 0
	tokenReviewCallCount := 0

	// Mock TokenReview
	fakeClient.PrependReactor("create", "tokenreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
		tokenReviewCallCount++
		return true, &authenticationv1.TokenReview{
			Status: authenticationv1.TokenReviewStatus{
				Authenticated: true,
				User: authenticationv1.UserInfo{
					Username: "test-user",
					UID:      "test-uid",
					Groups:   []string{"test-group"},
				},
			},
		}, nil
	})

	// Mock SubjectAccessReview - allow all and count calls
	fakeClient.PrependReactor("create", "subjectaccessreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
		sarCallCount++
		return true, &authv1.SubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: true,
			},
		}, nil
	})

	authorizer := newK8sAuthorizer(fakeClient)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set(stackroxNamespaceHeader, "test-namespace")

	// First request - should perform TokenReview and SAR checks
	userInfo, err := authorizer.authenticate(context.Background(), req)
	assert.NoError(t, err)
	err = authorizer.authorize(context.Background(), userInfo, req)
	assert.NoError(t, err)

	firstSARCallCount := sarCallCount
	firstTokenReviewCallCount := tokenReviewCallCount

	// Verify we made at least one SAR call on first authorization
	assert.Greater(t, firstSARCallCount, 0, "First authorization should perform at least one SAR call")

	// Second request with same token and namespace - should use cache
	userInfo, err = authorizer.authenticate(context.Background(), req)
	assert.NoError(t, err)
	err = authorizer.authorize(context.Background(), userInfo, req)
	assert.NoError(t, err)

	// Both SAR and TokenReview calls should NOT increase (all cached)
	assert.Equal(t, firstSARCallCount, sarCallCount, "Second authorization should use cached SAR results")
	assert.Equal(t, firstTokenReviewCallCount, tokenReviewCallCount, "TokenReview should use cached results")
}

func TestK8sAuthorizer_CachingBehavior_Denied(t *testing.T) {
	fakeClient := fake.NewClientset()

	sarCallCount := 0
	tokenReviewCallCount := 0

	// Mock TokenReview
	fakeClient.PrependReactor("create", "tokenreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
		tokenReviewCallCount++
		return true, &authenticationv1.TokenReview{
			Status: authenticationv1.TokenReviewStatus{
				Authenticated: true,
				User: authenticationv1.UserInfo{
					Username: "test-user",
					UID:      "test-uid",
					Groups:   []string{"test-group"},
				},
			},
		}, nil
	})

	// Mock SubjectAccessReview - deny all
	fakeClient.PrependReactor("create", "subjectaccessreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
		sarCallCount++
		return true, &authv1.SubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: false,
			},
		}, nil
	})

	authorizer := newK8sAuthorizer(fakeClient)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set(stackroxNamespaceHeader, "test-namespace")

	// First request - should perform TokenReview and SAR checks, then be denied
	userInfo, err := authorizer.authenticate(context.Background(), req)
	assert.NoError(t, err)
	err = authorizer.authorize(context.Background(), userInfo, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lacks")

	firstSARCallCount := sarCallCount
	firstTokenReviewCallCount := tokenReviewCallCount

	// Verify we made at least one SAR call on first authorization
	assert.Greater(t, firstSARCallCount, 0, "First authorization should perform at least one SAR call")

	// Second request with same token and namespace - should use cached denial
	userInfo, err = authorizer.authenticate(context.Background(), req)
	assert.NoError(t, err)
	err = authorizer.authorize(context.Background(), userInfo, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lacks")

	// Both SAR and TokenReview calls should NOT increase (all cached)
	assert.Equal(t, firstSARCallCount, sarCallCount, "Second authorization should use cached SAR denial results")
	assert.Equal(t, firstTokenReviewCallCount, tokenReviewCallCount, "TokenReview should use cached results")
}

func TestK8sAuthorizer_TokenReviewCaching(t *testing.T) {
	t.Run("successful TokenReview is cached", func(t *testing.T) {
		fakeClient := fake.NewClientset()
		tokenReviewCallCount := 0

		// Mock TokenReview to return authenticated
		fakeClient.PrependReactor("create", "tokenreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
			tokenReviewCallCount++
			return true, &authenticationv1.TokenReview{
				Status: authenticationv1.TokenReviewStatus{
					Authenticated: true,
					User: authenticationv1.UserInfo{
						Username: "test-user",
						Groups:   []string{"test-group"},
					},
				},
			}, nil
		})

		authorizer := newK8sAuthorizer(fakeClient)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		// First request
		_, err := authorizer.authenticate(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, 1, tokenReviewCallCount, "First request should perform TokenReview")

		// Second request with same token - should use cached TokenReview
		_, err = authorizer.authenticate(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, 1, tokenReviewCallCount, "Second request should use cached TokenReview")
	})

	t.Run("failed TokenReview is NOT cached", func(t *testing.T) {
		fakeClient := fake.NewClientset()
		tokenReviewCallCount := 0

		// Mock TokenReview to return unauthenticated
		fakeClient.PrependReactor("create", "tokenreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
			tokenReviewCallCount++
			return true, &authenticationv1.TokenReview{
				Status: authenticationv1.TokenReviewStatus{
					Authenticated: false,
				},
			}, nil
		})

		authorizer := newK8sAuthorizer(fakeClient)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		// First request - should fail
		_, err := authorizer.authenticate(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token authentication failed")
		assert.Equal(t, 1, tokenReviewCallCount, "First request should perform TokenReview")

		// Second request with same token - should perform TokenReview again (not cached)
		_, err = authorizer.authenticate(context.Background(), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token authentication failed")
		assert.Equal(t, 2, tokenReviewCallCount, "Second request should perform TokenReview again (failures not cached)")
	})
}

func TestK8sAuthorizer_TokenReviewCoalescing(t *testing.T) {
	t.Run("concurrent TokenReview requests for same token make only one API call", func(t *testing.T) {
		fakeClient := fake.NewClientset()
		var callCount atomic.Int32
		var wg sync.WaitGroup

		// Barrier to keep the TokenReview in-flight while goroutines start.
		// This ensures deterministic coalescing behavior without relying on timing.
		barrier := make(chan struct{})

		fakeClient.PrependReactor("create", "tokenreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
			callCount.Add(1)
			// Block until test signals all goroutines have started
			<-barrier
			return true, &authenticationv1.TokenReview{
				Status: authenticationv1.TokenReviewStatus{
					Authenticated: true,
					User: authenticationv1.UserInfo{
						Username: "test-user",
						UID:      "test-uid",
						Groups:   []string{"test-group"},
					},
				},
			}, nil
		})

		authorizer := newK8sAuthorizer(fakeClient)

		const numGoroutines = 10
		results := make([]*authenticationv1.UserInfo, numGoroutines)
		errs := make([]error, numGoroutines)

		// Use a separate WaitGroup to track when all goroutines have started
		var startWg sync.WaitGroup
		startWg.Add(numGoroutines)

		// Launch concurrent requests with the same token
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				startWg.Done() // Signal this goroutine has started
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Authorization", "Bearer same-token")
				results[idx], errs[idx] = authorizer.authenticate(context.Background(), req)
			}(i)
		}

		// Wait for all goroutines to start, then release the barrier
		startWg.Wait()
		close(barrier)
		wg.Wait()

		// Verify only ONE TokenReview API call was made
		assert.Equal(t, int32(1), callCount.Load(),
			"expected exactly 1 TokenReview call for %d concurrent requests", numGoroutines)

		// Verify all goroutines got the same result
		for i := 0; i < numGoroutines; i++ {
			require.NoError(t, errs[i])
			assert.Equal(t, "test-user", results[i].Username)
		}
	})

	t.Run("different tokens are not coalesced", func(t *testing.T) {
		fakeClient := fake.NewClientset()
		var callCount atomic.Int32

		fakeClient.PrependReactor("create", "tokenreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
			count := callCount.Add(1)
			return true, &authenticationv1.TokenReview{
				Status: authenticationv1.TokenReviewStatus{
					Authenticated: true,
					User: authenticationv1.UserInfo{
						Username: "user-" + string('0'+count),
					},
				},
			}, nil
		})

		authorizer := newK8sAuthorizer(fakeClient)

		var wg sync.WaitGroup
		wg.Add(3)

		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer token-a")
			_, _ = authorizer.authenticate(context.Background(), req)
		}()
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer token-b")
			_, _ = authorizer.authenticate(context.Background(), req)
		}()
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer token-c")
			_, _ = authorizer.authenticate(context.Background(), req)
		}()

		wg.Wait()

		// Should have made 3 separate TokenReview calls
		assert.Equal(t, int32(3), callCount.Load())
	})
}

func TestK8sAuthorizer_AuthorizationCoalescing(t *testing.T) {
	t.Run("concurrent authorization requests for same user/namespace make only one set of SAR calls", func(t *testing.T) {
		fakeClient := fake.NewClientset()
		var sarCallCount atomic.Int32
		var wg sync.WaitGroup

		// Barrier to keep the SAR in-flight while goroutines start.
		// This ensures deterministic coalescing behavior without relying on timing.
		barrier := make(chan struct{})

		fakeClient.PrependReactor("create", "subjectaccessreviews", func(action k8sTesting.Action) (bool, runtime.Object, error) {
			sarCallCount.Add(1)
			// Block until test signals all goroutines have started
			<-barrier
			return true, &authv1.SubjectAccessReview{
				Status: authv1.SubjectAccessReviewStatus{
					Allowed: true,
				},
			}, nil
		})

		// Create authorizer with only one resource to check for simpler test
		authorizer := &k8sAuthorizer{
			client:       fakeClient,
			tokenCache:   newK8sAuthorizer(fakeClient).tokenCache,
			authzCache:   newK8sAuthorizer(fakeClient).authzCache,
			authzFlight:  newK8sAuthorizer(fakeClient).authzFlight,
			verbsToCheck: []string{"get"},
			resourcesToCheck: []k8sResource{
				{Resource: "pods", Group: ""},
			},
		}

		userInfo := &authenticationv1.UserInfo{
			Username: "test-user",
			UID:      "test-uid",
			Groups:   []string{"test-group"},
		}

		const numGoroutines = 10
		errs := make([]error, numGoroutines)

		// Use a separate WaitGroup to track when all goroutines have started
		var startWg sync.WaitGroup
		startWg.Add(numGoroutines)

		// Launch concurrent authorize requests for the same user/namespace
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				startWg.Done() // Signal this goroutine has started
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set(stackroxNamespaceHeader, "test-namespace")
				errs[idx] = authorizer.authorize(context.Background(), userInfo, req)
			}(i)
		}

		// Wait for all goroutines to start, then release the barrier
		startWg.Wait()
		close(barrier)
		wg.Wait()

		// Verify only ONE SAR API call was made (1 resource × 1 verb)
		// All concurrent requests share the same singleflight execution.
		assert.Equal(t, int32(1), sarCallCount.Load(),
			"expected exactly 1 SAR call for %d concurrent requests", numGoroutines)

		// Verify all goroutines got success
		for i := 0; i < numGoroutines; i++ {
			assert.NoError(t, errs[i])
		}
	})
}
