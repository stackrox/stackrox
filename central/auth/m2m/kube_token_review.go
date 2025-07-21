package m2m

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	v1 "k8s.io/api/authentication/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// kubeTokenReviewVerifier verifies tokens using the Kubernetes TokenReview API.
type kubeTokenReviewVerifier struct {
	clientset kubernetes.Interface
}

func (k *kubeTokenReviewVerifier) VerifyIDToken(ctx context.Context, rawIDToken string) (*IDToken, error) {
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

/*
	resp, err := k.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "performing TokenReview request")
	}
	defer utils.IgnoreError(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("TokenReview API returned status %d", resp.StatusCode)
	}

	var trResp struct {
		Status struct {
			Authenticated bool `json:"authenticated"`
			User          struct {
				Username string   `json:"username"`
				UID      string   `json:"uid"`
				Groups   []string `json:"groups"`
			} `json:"user"`
			Audiences []string `json:"audiences"`
			Error     string   `json:"error"`
		} `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&trResp); err != nil {
		return nil, errors.Wrap(err, "decoding TokenReview response")
	}
	if !trResp.Status.Authenticated {
		return nil, errors.Errorf("token not authenticated: %s", trResp.Status.Error)
	}

	// Construct a minimal oidc.IDToken with user info in claims
	claims := map[string]interface{}{
		"sub":    trResp.Status.User.UID,
		"name":   trResp.Status.User.Username,
		"groups": trResp.Status.User.Groups,
	}
	rawClaims, _ := json.Marshal(claims)

	token := &IDToken{
		Subject: trResp.Status.User.UID,
		Claims: func(v interface{}) error {
			return json.Unmarshal(rawClaims, v)
		},
		Audience: trResp.Status.Audiences,
	}
	return token, nil
}
*/
