package certrefresh

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"os"

	"github.com/pkg/errors"
	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/pods"
	"github.com/stackrox/rox/sensor/utils"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const (
	tlsCABundleAnnotationKey  = "stackrox.io/info"
	tlsCABundleAnnotationText = "ConfigMap containing TLS CA certificates trusted by Central. Automatically generated - do not modify."
)

var (
	caBundleLog = logging.LoggerForModule()
)

func CreateTLSCABundleConfigMapFromCerts(ctx context.Context, certs []*x509.Certificate, k8sClient kubernetes.Interface) error {
	namespace := pods.GetPodNamespace()
	podName := os.Getenv("POD_NAME")

	ownerRef, err := FetchSensorDeploymentOwnerRef(ctx, podName, namespace, k8sClient, wait.Backoff{})
	if err != nil {
		return errors.Wrap(err, "failed to fetch sensor deployment owner reference")
	}

	data, err := createCABundleData(certs)
	if err != nil {
		return errors.Wrap(err, "failed to create ConfigMap data")
	}

	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pkgKubernetes.TLSCABundleConfigMapName,
			Namespace: namespace,
			Annotations: map[string]string{
				tlsCABundleAnnotationKey: tlsCABundleAnnotationText,
			},
			Labels: func() map[string]string {
				labels := utils.GetSensorKubernetesLabels()
				// This label is required by the Operator in order to cache the ConfigMap.
				labels["app.stackrox.io/managed-by"] = "sensor"
				return labels
			}(),
			OwnerReferences: []metav1.OwnerReference{*ownerRef},
		},
		Data: data,
	}

	_, err = k8sClient.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "failed to create TLS CA bundle ConfigMap")
		}

		_, err = k8sClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to update TLS CA bundle ConfigMap")
		}
		caBundleLog.Infof("Updated TLS CA bundle ConfigMap %s/%s", namespace, pkgKubernetes.TLSCABundleConfigMapName)
	} else {
		caBundleLog.Infof("Created TLS CA bundle ConfigMap %s/%s", namespace, pkgKubernetes.TLSCABundleConfigMapName)
	}

	return nil
}

func createCABundleData(certs []*x509.Certificate) (map[string]string, error) {
	if len(certs) == 0 {
		return nil, errors.New("no certificates provided")
	}

	data := make(map[string]string)
	var allCertsPEM []byte
	for _, cert := range certs {
		certPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		allCertsPEM = append(allCertsPEM, certPEM...)
	}

	data[pkgKubernetes.TLSCABundleKey] = string(allCertsPEM)
	caBundleLog.Infof("Created CA bundle with %d certificates", len(certs))
	return data, nil
}
