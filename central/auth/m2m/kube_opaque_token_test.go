package m2m

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestKubeOpaqueTokenVerifier_VerifyIDToken_Success(t *testing.T) {
	rawToken := "raw-token-value"
	// Prepare fake clientset and reactor for TokenReview create.
	fakeClient := fake.NewSimpleClientset()
	fakeClient.Fake.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		// Simulate successful TokenReview with authenticated status.
		resp := &v1.TokenReview{
			Status: v1.TokenReviewStatus{
				Authenticated: true,
				User: v1.UserInfo{
					UID:      "uid123",
					Username: "user123",
					Groups:   []string{"group1", "group2"},
				},
				Audiences: []string{"aud1", "aud2"},
			},
		}
		return true, resp, nil
	})

	verifier := kubeOpaqueTokenVerifier{clientset: fakeClient}
	token, err := verifier.VerifyIDToken(context.Background(), rawToken)
	require.NoError(t, err)
	require.NotNil(t, token)
	// Verify core fields.
	require.Equal(t, "uid123", token.Subject)
	require.Equal(t, []string{"aud1", "aud2"}, token.Audience)

	// Verify claims unmarshalling.
	type claimsStruct struct {
		Sub    string   `json:"sub"`
		Name   string   `json:"name"`
		Groups []string `json:"groups"`
	}
	var claims claimsStruct
	err = token.Claims(&claims)
	require.NoError(t, err)
	require.Equal(t, "uid123", claims.Sub)
	require.Equal(t, "user123", claims.Name)
	require.Equal(t, []string{"group1", "group2"}, claims.Groups)
}

func TestKubeOpaqueTokenVerifier_VerifyIDToken_CreateError(t *testing.T) {
	rawToken := "raw-token-value"
	fakeClient := fake.NewSimpleClientset()
	fakeClient.Fake.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("create failed")
	})
	verifier := kubeOpaqueTokenVerifier{clientset: fakeClient}
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
	verifier := kubeOpaqueTokenVerifier{clientset: fakeClient}
	token, err := verifier.VerifyIDToken(context.Background(), rawToken)
	require.Nil(t, token)
	require.Error(t, err)
	require.Contains(t, err.Error(), "token not authenticated: invalid token")
}
