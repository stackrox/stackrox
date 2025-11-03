package m2m

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"k8s.io/client-go/rest"
)

var (
	serviceAccountIssuer string
	getIssuerOnce        sync.Once
)

// GetKubernetesIssuerOrEmpty returns the kubernetes token issuer or an empty
// string if the issuer could not be identified.
func GetKubernetesIssuerOrEmpty() string {
	getIssuerOnce.Do(func() {
		issuer, err := getKubernetesIssuer()
		if err != nil {
			log.Errorf("could not read service account issuer: %v", err)
			return
		}
		serviceAccountIssuer = issuer
	})
	return serviceAccountIssuer
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
