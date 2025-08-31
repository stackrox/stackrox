package m2m

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	v1 "k8s.io/api/authentication/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// kubeOpaqueTokenVerifier verifies tokens using the Kubernetes TokenReview API.
type kubeOpaqueTokenVerifier struct {
	clientset kubernetes.Interface
}

func (k *kubeOpaqueTokenVerifier) VerifyIDToken(ctx context.Context, rawIDToken string) (*IDToken, error) {
	tr := &v1.TokenReview{
		Spec: v1.TokenReviewSpec{
			Token: rawIDToken,
		},
	}
	trResp, err := k.clientset.AuthenticationV1().TokenReviews().
		Create(ctx, tr, metaV1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "performing TokenReview request")
	}
	if !trResp.Status.Authenticated {
		return nil, errors.Errorf("token not authenticated: %s", trResp.Status.Error)
	}

	claims := map[string]any{
		"sub":    trResp.Status.User.UID,
		"name":   trResp.Status.User.Username,
		"groups": trResp.Status.User.Groups,
	}
	rawClaims, _ := json.Marshal(claims)

	token := &IDToken{
		Subject: trResp.Status.User.UID,
		Claims: func(v any) error {
			return json.Unmarshal(rawClaims, v)
		},
		Audience: trResp.Status.Audiences,
	}
	return token, nil
}
