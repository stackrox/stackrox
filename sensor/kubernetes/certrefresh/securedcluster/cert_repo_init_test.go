package securedcluster

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/securedcluster"
	"github.com/stretchr/testify/assert"
	appsApiv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	namespace        = "stackrox-ns"
	sensorDeployment = &appsApiv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sensor-deployment",
			Namespace: namespace,
		},
	}
	securedClusterCertificateSet = &storage.TypedServiceCertificateSet{
		CaPem: make([]byte, 2),
		ServiceCerts: []*storage.TypedServiceCertificate{
			createServiceCertificate(storage.ServiceType_SENSOR_SERVICE),
			createServiceCertificate(storage.ServiceType_ADMISSION_CONTROL_SERVICE),
			createServiceCertificate(storage.ServiceType_COLLECTOR_SERVICE),
			createServiceCertificate(storage.ServiceType_SCANNER_SERVICE),
			createServiceCertificate(storage.ServiceType_SCANNER_DB_SERVICE),
			createServiceCertificate(storage.ServiceType_SCANNER_V4_INDEXER_SERVICE),
			createServiceCertificate(storage.ServiceType_SCANNER_V4_DB_SERVICE),
		},
	}
)

func TestEnsureServiceCertificates(t *testing.T) {
	clientSet := fake.NewSimpleClientset(sensorDeployment)
	secretsClient := clientSet.CoreV1().Secrets(namespace)
	ctx := context.Background()

	repo := NewServiceCertificatesRepo(sensorOwnerReference(sensorDeployment), namespace, secretsClient)
	persistedCertificates, err := repo.EnsureServiceCertificates(ctx, securedClusterCertificateSet)

	assert.NoError(t, err)
	protoassert.SlicesEqual(t, securedClusterCertificateSet.GetServiceCerts(), persistedCertificates)

	expectedSecretNames := []string{
		securedcluster.SensorTLSSecretName,
		securedcluster.CollectorTLSSecretName,
		securedcluster.AdmissionControlTLSSecretName,
		securedcluster.ScannerTLSSecretName,
		securedcluster.ScannerDbTLSSecretName,
		securedcluster.ScannerV4IndexerTLSSecretName,
		securedcluster.ScannerV4DbTLSSecretName,
	}
	assert.Equal(t, len(expectedSecretNames), len(persistedCertificates))

	for _, secretName := range expectedSecretNames {
		_, err = secretsClient.Get(ctx, secretName, metav1.GetOptions{})
		assert.NoError(t, err)
	}
}

func sensorOwnerReference(sensorDeployment *appsApiv1.Deployment) metav1.OwnerReference {
	sensorDeploymentGVK := sensorDeployment.GroupVersionKind()
	blockOwnerDeletion := false
	isController := false
	return metav1.OwnerReference{
		APIVersion:         sensorDeploymentGVK.GroupVersion().String(),
		Kind:               sensorDeploymentGVK.Kind,
		Name:               sensorDeployment.GetName(),
		UID:                sensorDeployment.GetUID(),
		BlockOwnerDeletion: &blockOwnerDeletion,
		Controller:         &isController,
	}
}

func createServiceCertificate(serviceType storage.ServiceType) *storage.TypedServiceCertificate {
	return &storage.TypedServiceCertificate{
		ServiceType: serviceType,
		Cert: &storage.ServiceCertificate{
			CertPem: make([]byte, 0),
			KeyPem:  make([]byte, 1),
		},
	}
}
