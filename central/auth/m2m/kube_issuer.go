package m2m

import (
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
)

const k8sSATokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token" //#nosec G101 -- This is a false positive

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
	token, err := readK8sSAToken()
	if err != nil {
		return "", errors.Wrap(err, "failed to read service account token")
	}
	return IssuerFromRawIDToken(string(token))
}

func readK8sSAToken() ([]byte, error) {
	return os.ReadFile(k8sSATokenFile)
}
