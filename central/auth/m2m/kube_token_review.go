package m2m

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

// kubeTokenReviewVerifier verifies tokens using the Kubernetes TokenReview API.
type kubeTokenReviewVerifier struct {
	apiServer string
	client    *http.Client
}

// tokenReviewRequest represents the payload for the Kubernetes TokenReview API.
type tokenReviewRequest struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Spec       struct {
		Token string `json:"token"`
	} `json:"spec"`
}

func (k *kubeTokenReviewVerifier) VerifyIDToken(ctx context.Context, rawIDToken string) (*IDToken, error) {
	// Prepare TokenReview request
	tr := tokenReviewRequest{
		"authentication.k8s.io/v1",
		"TokenReview",
		struct {
			Token string `json:"token"`
		}{rawIDToken},
	}
	reqBody, _ := json.Marshal(tr)
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/apis/authentication.k8s.io/v1/tokenreviews",
		k.apiServer),
		bytes.NewReader(reqBody))
	if err != nil {
		return nil, errors.Wrap(err, "creating TokenReview request")
	}
	req.Header.Set("Content-Type", "application/json")

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
		/*
			Issuer:   k.apiServer,
			Expiry:   time.Now().Add(5 * time.Minute), // TokenReview doesn't provide expiry, so set a short one.
			IssuedAt: time.Now(),
		},*/
		Claims: func(v interface{}) error {
			return json.Unmarshal(rawClaims, v)
		},
		Audience: trResp.Status.Audiences,
	}
	return token, nil
}
