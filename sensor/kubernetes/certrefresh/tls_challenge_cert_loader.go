package certrefresh

import (
	"context"
	"crypto/x509"
	"os"

	"github.com/stackrox/rox/pkg/pods"
	"github.com/stackrox/rox/sensor/common/centralclient"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// TLSChallengeCertLoader returns a centralclient.CertLoader that:
// - performs the TLS challenge with Central and retrieves its trusted certificates
// - ceates/updates CA bundle ConfigMap for Admission Control's ValidatingWebhookConfiguration
// This is used for Operator managed clusters to enable CA rotation.
func TLSChallengeCertLoader(centralClient *centralclient.Client, k8sClient kubernetes.Interface) centralclient.CertLoader {
	return func() []*x509.Certificate {
		ctx := context.Background()
		certs, centralCAs, err := centralClient.GetTLSTrustedCerts(ctx)
		if err != nil {
			// only logs errors to not break Sensor start-up.
			log.Errorf("\n#------------------------------------------------------------------------------\n"+
				"# Failed to fetch centrals TLS certs: %v\n"+
				"#------------------------------------------------------------------------------", err)
		} else if len(centralCAs) > 0 {
			log.Debug("Updating TLS CA bundle ConfigMap from TLSChallenge")
			handleCABundleConfigMapUpdate(ctx, centralCAs, k8sClient)
		}
		return certs
	}
}

func handleCABundleConfigMapUpdate(ctx context.Context, centralCAs []*x509.Certificate, k8sClient kubernetes.Interface) {
	namespace := pods.GetPodNamespace()
	podName := os.Getenv("POD_NAME")
	ownerRef, err := FetchSensorDeploymentOwnerRef(ctx, podName, namespace,
		k8sClient, wait.Backoff{})
	if err != nil {
		log.Warnf("Failed to fetch sensor deployment owner reference: %v", err)
		ownerRef = nil
	}

	if err := CreateTLSCABundleConfigMapFromCerts(ctx, centralCAs,
		k8sClient.CoreV1().ConfigMaps(namespace), ownerRef); err != nil {
		log.Warnf("Failed to create/update TLS CA bundle ConfigMap: %v", err)
	}
}
