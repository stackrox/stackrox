package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authnv1 "k8s.io/api/authentication/v1"
	authzv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestK8sValidator_ValidateDeploymentAccess_Success(t *testing.T) {
	ctx := context.Background()
	k8sClient := fake.NewSimpleClientset()

	// Mock TokenReview response
	k8sClient.Fake.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		tr := createAction.GetObject().(*authnv1.TokenReview)

		// Return authenticated user
		tr.Status = authnv1.TokenReviewStatus{
			Authenticated: true,
			User: authnv1.UserInfo{
				Username: "test-user@example.com",
				UID:      "user-123",
				Groups:   []string{"system:authenticated", "developers"},
			},
		}
		return true, tr, nil
	})

	// Mock SubjectAccessReview response
	k8sClient.Fake.PrependReactor("create", "subjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		sar := createAction.GetObject().(*authzv1.SubjectAccessReview)

		// Grant access
		sar.Status = authzv1.SubjectAccessReviewStatus{
			Allowed: true,
		}
		return true, sar, nil
	})

	validator := NewK8sValidator(k8sClient)
	userInfo, err := validator.ValidateDeploymentAccess(ctx, "valid-token", "default", "nginx")

	require.NoError(t, err)
	assert.Equal(t, "test-user@example.com", userInfo.Username)
	assert.Equal(t, "user-123", userInfo.UID)
	assert.Contains(t, userInfo.Groups, "developers")
}

func TestK8sValidator_ValidateDeploymentAccess_TokenNotAuthenticated(t *testing.T) {
	ctx := context.Background()
	k8sClient := fake.NewSimpleClientset()

	// Mock TokenReview response - not authenticated
	k8sClient.Fake.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		tr := createAction.GetObject().(*authnv1.TokenReview)

		tr.Status = authnv1.TokenReviewStatus{
			Authenticated: false,
		}
		return true, tr, nil
	})

	validator := NewK8sValidator(k8sClient)
	userInfo, err := validator.ValidateDeploymentAccess(ctx, "invalid-token", "default", "nginx")

	require.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "invalid or expired token")
}

func TestK8sValidator_ValidateDeploymentAccess_RBACDenied(t *testing.T) {
	ctx := context.Background()
	k8sClient := fake.NewSimpleClientset()

	// Mock TokenReview response
	k8sClient.Fake.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		tr := createAction.GetObject().(*authnv1.TokenReview)

		tr.Status = authnv1.TokenReviewStatus{
			Authenticated: true,
			User: authnv1.UserInfo{
				Username: "limited-user",
				UID:      "user-456",
				Groups:   []string{"system:authenticated"},
			},
		}
		return true, tr, nil
	})

	// Mock SubjectAccessReview response - deny access
	k8sClient.Fake.PrependReactor("create", "subjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		sar := createAction.GetObject().(*authzv1.SubjectAccessReview)

		sar.Status = authzv1.SubjectAccessReviewStatus{
			Allowed: false,
			Reason:  "user does not have permission to access deployments in namespace default",
		}
		return true, sar, nil
	})

	validator := NewK8sValidator(k8sClient)
	userInfo, err := validator.ValidateDeploymentAccess(ctx, "valid-token", "default", "nginx")

	require.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "not have permission")
}

func TestK8sValidator_ValidateDeploymentAccess_EmptyDeployment(t *testing.T) {
	ctx := context.Background()
	k8sClient := fake.NewSimpleClientset()

	var capturedSAR *authzv1.SubjectAccessReview

	// Mock TokenReview response
	k8sClient.Fake.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		tr := createAction.GetObject().(*authnv1.TokenReview)

		tr.Status = authnv1.TokenReviewStatus{
			Authenticated: true,
			User: authnv1.UserInfo{
				Username: "test-user",
				UID:      "user-789",
				Groups:   []string{"system:authenticated"},
			},
		}
		return true, tr, nil
	})

	// Mock SubjectAccessReview response
	k8sClient.Fake.PrependReactor("create", "subjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		sar := createAction.GetObject().(*authzv1.SubjectAccessReview)
		capturedSAR = sar.DeepCopy()

		sar.Status = authzv1.SubjectAccessReviewStatus{
			Allowed: true,
		}
		return true, sar, nil
	})

	validator := NewK8sValidator(k8sClient)
	_, err := validator.ValidateDeploymentAccess(ctx, "valid-token", "production", "")

	require.NoError(t, err)

	// Verify SubjectAccessReview was created without deployment name
	assert.Equal(t, "production", capturedSAR.Spec.ResourceAttributes.Namespace)
	assert.Equal(t, "", capturedSAR.Spec.ResourceAttributes.Name)
	assert.Equal(t, "deployments", capturedSAR.Spec.ResourceAttributes.Resource)
}

func TestK8sValidator_ValidateDeploymentAccess_EmptyNamespace(t *testing.T) {
	ctx := context.Background()
	k8sClient := fake.NewSimpleClientset()

	var capturedSAR *authzv1.SubjectAccessReview

	// Mock TokenReview response
	k8sClient.Fake.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		tr := createAction.GetObject().(*authnv1.TokenReview)

		tr.Status = authnv1.TokenReviewStatus{
			Authenticated: true,
			User: authnv1.UserInfo{
				Username: "admin-user",
				UID:      "user-admin",
				Groups:   []string{"system:masters"},
			},
		}
		return true, tr, nil
	})

	// Mock SubjectAccessReview response
	k8sClient.Fake.PrependReactor("create", "subjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		sar := createAction.GetObject().(*authzv1.SubjectAccessReview)
		capturedSAR = sar.DeepCopy()

		sar.Status = authzv1.SubjectAccessReviewStatus{
			Allowed: true,
		}
		return true, sar, nil
	})

	validator := NewK8sValidator(k8sClient)
	_, err := validator.ValidateDeploymentAccess(ctx, "valid-token", "", "")

	require.NoError(t, err)

	// Verify SubjectAccessReview was created for cluster-wide access
	assert.Equal(t, "", capturedSAR.Spec.ResourceAttributes.Namespace)
	assert.Equal(t, "", capturedSAR.Spec.ResourceAttributes.Name)
	assert.Equal(t, "deployments", capturedSAR.Spec.ResourceAttributes.Resource)
	assert.Equal(t, "apps", capturedSAR.Spec.ResourceAttributes.Group)
}

func TestK8sValidator_ValidateDeploymentAccess_TokenReviewError(t *testing.T) {
	ctx := context.Background()
	k8sClient := fake.NewSimpleClientset()

	// Mock TokenReview error
	k8sClient.Fake.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("API server unavailable")
	})

	validator := NewK8sValidator(k8sClient)
	userInfo, err := validator.ValidateDeploymentAccess(ctx, "token", "default", "nginx")

	require.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "invalid or expired token")
}

func TestK8sValidator_ValidateDeploymentAccess_SubjectAccessReviewError(t *testing.T) {
	ctx := context.Background()
	k8sClient := fake.NewSimpleClientset()

	// Mock TokenReview response
	k8sClient.Fake.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		tr := createAction.GetObject().(*authnv1.TokenReview)

		tr.Status = authnv1.TokenReviewStatus{
			Authenticated: true,
			User: authnv1.UserInfo{
				Username: "test-user",
				UID:      "user-123",
				Groups:   []string{"system:authenticated"},
			},
		}
		return true, tr, nil
	})

	// Mock SubjectAccessReview error
	k8sClient.Fake.PrependReactor("create", "subjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("authorization API error")
	})

	validator := NewK8sValidator(k8sClient)
	userInfo, err := validator.ValidateDeploymentAccess(ctx, "token", "default", "nginx")

	require.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "authorization check failed")
}

func TestK8sValidator_ValidateDeploymentAccess_EmptyUsername(t *testing.T) {
	ctx := context.Background()
	k8sClient := fake.NewSimpleClientset()

	// Mock TokenReview response with empty username
	k8sClient.Fake.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		tr := createAction.GetObject().(*authnv1.TokenReview)

		tr.Status = authnv1.TokenReviewStatus{
			Authenticated: true,
			User: authnv1.UserInfo{
				Username: "", // Empty username
				UID:      "user-123",
				Groups:   []string{"system:authenticated"},
			},
		}
		return true, tr, nil
	})

	validator := NewK8sValidator(k8sClient)
	userInfo, err := validator.ValidateDeploymentAccess(ctx, "token", "default", "nginx")

	require.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "invalid or expired token")
}

func TestK8sValidator_ValidateDeploymentAccess_VerifySARRequest(t *testing.T) {
	ctx := context.Background()
	k8sClient := fake.NewSimpleClientset()

	var capturedSAR *authzv1.SubjectAccessReview

	// Mock TokenReview response
	k8sClient.Fake.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		tr := createAction.GetObject().(*authnv1.TokenReview)

		tr.Status = authnv1.TokenReviewStatus{
			Authenticated: true,
			User: authnv1.UserInfo{
				Username: "test-user",
				UID:      "user-123",
				Groups:   []string{"developers", "system:authenticated"},
			},
		}
		return true, tr, nil
	})

	// Mock SubjectAccessReview response and capture request
	k8sClient.Fake.PrependReactor("create", "subjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		sar := createAction.GetObject().(*authzv1.SubjectAccessReview)
		capturedSAR = sar.DeepCopy()

		sar.Status = authzv1.SubjectAccessReviewStatus{
			Allowed: true,
		}
		return true, sar, nil
	})

	validator := NewK8sValidator(k8sClient)
	_, err := validator.ValidateDeploymentAccess(ctx, "token", "staging", "api-server")

	require.NoError(t, err)

	// Verify SubjectAccessReview request was constructed correctly
	assert.Equal(t, "test-user", capturedSAR.Spec.User)
	assert.Equal(t, "user-123", capturedSAR.Spec.UID)
	assert.ElementsMatch(t, []string{"developers", "system:authenticated"}, capturedSAR.Spec.Groups)
	assert.Equal(t, "staging", capturedSAR.Spec.ResourceAttributes.Namespace)
	assert.Equal(t, "api-server", capturedSAR.Spec.ResourceAttributes.Name)
	assert.Equal(t, "get", capturedSAR.Spec.ResourceAttributes.Verb)
	assert.Equal(t, "apps", capturedSAR.Spec.ResourceAttributes.Group)
	assert.Equal(t, "deployments", capturedSAR.Spec.ResourceAttributes.Resource)
}
