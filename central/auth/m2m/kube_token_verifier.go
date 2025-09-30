package m2m

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/utils"
	v1 "k8s.io/api/authentication/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// #nosec G101 -- This is a false positive.
const KubernetesDefaultSvcTokenIssuer = "https://kubernetes.default.svc"

// kubeTokenVerifier verifies tokens using the Kubernetes TokenReview API.
type kubeTokenVerifier struct {
	clientset kubernetes.Interface
}

func newKubeTokenVerifier() (*kubeTokenVerifier, error) {
	cfg, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		return nil, err
	}
	c, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &kubeTokenVerifier{c}, nil
}

// getKubernetesIssuer discovers the kubernetes token issuer.
func getKubernetesIssuer() (string, error) {
	cfg, err := k8sutil.GetK8sInClusterConfig()
	if err != nil {
		return "", errors.Wrap(err, "could not get k8s in cluster configuration")
	}

	discoveryURL := fmt.Sprintf("%s/.well-known/openid-configuration",
		strings.TrimSuffix(cfg.Host, "/"))

	tr, err := rest.TransportFor(cfg)
	if err != nil {
		return "", errors.Wrap(err, "could not create transport")
	}

	client := http.Client{Transport: tr}
	resp, err := client.Get(discoveryURL)
	if err != nil {
		return "", errors.Wrapf(err, "request to %q failed", discoveryURL)
	}
	defer utils.IgnoreError(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		return "", httputil.NewError(resp.StatusCode, resp.Status)
	}

	var discovery struct {
		Issuer string `json:"issuer"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return "", errors.Wrap(err, "failed to decode discovery document")
	}

	return discovery.Issuer, nil
}

func (k *kubeTokenVerifier) VerifyIDToken(ctx context.Context, rawIDToken string) (*IDToken, error) {
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
	return tokenFromReview(&trResp.Status), nil
}

func tokenFromReview(trs *v1.TokenReviewStatus) *IDToken {
	return &IDToken{
		Subject: trs.User.Username,
		Claims: func(v any) error {
			trsPtr, ok := (v).(*v1.TokenReviewStatus)
			if !ok {
				return errox.InvariantViolation.New("unexpected claims unmarshalling request")
			}
			*trsPtr = *trs
			return nil
		},
		Audience: trs.Audiences,
	}
}
