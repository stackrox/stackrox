package m2m

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestKubeTokenVerifier_VerifyIDToken_Success(t *testing.T) {
	rawToken := "raw-token-value"
	// Prepare fake clientset and reactor for TokenReview create.
	fakeClient := fake.NewSimpleClientset()
	status := v1.TokenReviewStatus{
		Authenticated: true,
		User: v1.UserInfo{
			UID:      "uid123",
			Username: "user123",
			Groups:   []string{"group1", "group2"},
		},
		Audiences: []string{"aud1", "aud2"},
	}
	fakeClient.Fake.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		// Simulate successful TokenReview with authenticated status.
		resp := &v1.TokenReview{
			Status: status,
		}
		return true, resp, nil
	})

	verifier := kubeTokenVerifier{clientset: fakeClient}
	token, err := verifier.VerifyIDToken(context.Background(), rawToken)
	require.NoError(t, err)
	require.NotNil(t, token)
	// Verify core fields.
	require.Equal(t, "user123", token.Subject)
	require.Equal(t, []string{"aud1", "aud2"}, token.Audience)

	// Verify claims unmarshalling.
	var claims map[string]any
	err = token.Claims(&claims)
	require.ErrorIs(t, err, errox.InvariantViolation)

	var trs v1.TokenReviewStatus
	err = token.Claims(&trs)
	require.NoError(t, err)
	require.Equal(t, status, trs)
}

func TestKubeOpaqueTokenVerifier_VerifyIDToken_CreateError(t *testing.T) {
	rawToken := "raw-token-value"
	fakeClient := fake.NewSimpleClientset()
	fakeClient.Fake.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("create failed")
	})
	verifier := kubeTokenVerifier{clientset: fakeClient}
	token, err := verifier.VerifyIDToken(context.Background(), rawToken)
	require.Nil(t, token)
	require.Error(t, err)
	require.Contains(t, err.Error(), "performing TokenReview request")
	require.Contains(t, err.Error(), "create failed")
}

func TestKubeOpaqueTokenVerifier_VerifyIDToken_NotAuthenticated(t *testing.T) {
	rawToken := "raw-token-value"
	fakeClient := fake.NewSimpleClientset()
	fakeClient.Fake.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		// Simulate unauthenticated token review
		resp := &v1.TokenReview{
			Status: v1.TokenReviewStatus{
				Authenticated: false,
				Error:         "invalid token",
			},
		}
		return true, resp, nil
	})
	verifier := kubeTokenVerifier{clientset: fakeClient}
	token, err := verifier.VerifyIDToken(context.Background(), rawToken)
	require.Nil(t, token)
	require.Error(t, err)
	require.Contains(t, err.Error(), "token not authenticated: invalid token")
}
