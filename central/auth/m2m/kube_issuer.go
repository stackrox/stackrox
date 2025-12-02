package m2m

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"k8s.io/client-go/rest"
)

const k8sSATokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"

var (
	serviceAccountIssuer      string
	serviceAccountIssuerError error
	getIssuerOnce             sync.Once
)

// SetKubernetesIssuerForTest is used for testing the datastore without reading
// the in-cluster Kubernetes configuration.
func SetKubernetesIssuerForTest(_ *testing.T, issuer string) {
	getIssuerOnce.Do(func() {
		serviceAccountIssuer = issuer
	})
}

// GetKubernetesIssuer discovers the kubernetes token issuer.
func GetKubernetesIssuer() (string, error) {
	getIssuerOnce.Do(func() {
		serviceAccountIssuer, serviceAccountIssuerError = getKubernetesIssuer()
		if serviceAccountIssuerError != nil {
			log.Errorf("could not read service account issuer: %v",
				serviceAccountIssuerError)
		}
	})
	return serviceAccountIssuer, serviceAccountIssuerError
}

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

func k8sSATokenReader() ([]byte, error) {
	return os.ReadFile(k8sSATokenFile)
}
