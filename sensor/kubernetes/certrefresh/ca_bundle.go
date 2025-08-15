package certrefresh

import (
	"context"
	"crypto/x509"
	"encoding/pem"

	"github.com/pkg/errors"
	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	commonLabels "github.com/stackrox/rox/pkg/labels"
	"github.com/stackrox/rox/pkg/pods"
	"github.com/stackrox/rox/sensor/utils"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	tlsCABundleAnnotationKey  = "stackrox.io/info"
	tlsCABundleAnnotationText = "ConfigMap containing TLS CA certificates trusted by Central. Automatically generated - do not modify."
)

// CreateTLSCABundleConfigMapFromCerts creates or updates the TLS CA bundle ConfigMap from x509 certificates.
func CreateTLSCABundleConfigMapFromCerts(ctx context.Context, certs []*x509.Certificate, configMapClient corev1.ConfigMapInterface, ownerRef *metav1.OwnerReference) error {
	pemData, err := convertCertsToPEM(certs)
	if err != nil {
		return errors.Wrap(err, "failed to convert certificates to PEM")
	}
	return CreateTLSCABundleConfigMapFromPEM(ctx, pemData, configMapClient, ownerRef)
}

// CreateTLSCABundleConfigMapFromPEM creates or updates the TLS CA bundle ConfigMap from PEM data.
func CreateTLSCABundleConfigMapFromPEM(ctx context.Context, pemData []byte, configMapClient corev1.ConfigMapInterface, ownerRef *metav1.OwnerReference) error {
	if len(pemData) == 0 {
		return errors.New("no PEM data provided")
	}

	return createOrUpdateCABundleConfigMap(ctx, pemData, configMapClient, ownerRef)
}

func createOrUpdateCABundleConfigMap(ctx context.Context, pemData []byte, configMapClient corev1.ConfigMapInterface, ownerRef *metav1.OwnerReference) error {
	namespace := pods.GetPodNamespace()

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
				labels[commonLabels.ManagedByLabelKey] = commonLabels.ManagedBySensor
				return labels
			}(),
		},
		Data: map[string]string{
			pkgKubernetes.TLSCABundleKey: string(pemData),
		},
	}

	// Add owner reference if provided
	if ownerRef != nil {
		configMap.ObjectMeta.OwnerReferences = []metav1.OwnerReference{*ownerRef}
	}

	_, err := configMapClient.Create(ctx, configMap, metav1.CreateOptions{})
	if err == nil {
		log.Debugf("Created TLS CA bundle ConfigMap %s/%s", namespace, pkgKubernetes.TLSCABundleConfigMapName)
		return nil
	}

	if !k8serrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to create TLS CA bundle ConfigMap")
	}

	_, err = configMapClient.Update(ctx, configMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update TLS CA bundle ConfigMap")
	}
	log.Debugf("Updated TLS CA bundle ConfigMap %s/%s", namespace, pkgKubernetes.TLSCABundleConfigMapName)
	return nil
}

func convertCertsToPEM(certs []*x509.Certificate) ([]byte, error) {
	if len(certs) == 0 {
		return nil, errors.New("no certificates provided")
	}

	var allCertsPEM []byte
	for _, cert := range certs {
		certPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		allCertsPEM = append(allCertsPEM, certPEM...)
	}

	log.Debugf("Created CA bundle with %d certificates", len(certs))
	return allCertsPEM, nil
}
