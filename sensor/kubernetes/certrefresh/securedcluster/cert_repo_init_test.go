package securedcluster

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/testutils"
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
			testutils.CreateServiceCertificate(storage.ServiceType_SENSOR_SERVICE),
			testutils.CreateServiceCertificate(storage.ServiceType_ADMISSION_CONTROL_SERVICE),
			testutils.CreateServiceCertificate(storage.ServiceType_COLLECTOR_SERVICE),
			testutils.CreateServiceCertificate(storage.ServiceType_SCANNER_SERVICE),
			testutils.CreateServiceCertificate(storage.ServiceType_SCANNER_DB_SERVICE),
			testutils.CreateServiceCertificate(storage.ServiceType_SCANNER_V4_INDEXER_SERVICE),
			testutils.CreateServiceCertificate(storage.ServiceType_SCANNER_V4_DB_SERVICE),
		},
	}
)

func TestEnsureServiceCertificates(t *testing.T) {
	clientSet := fake.NewSimpleClientset(sensorDeployment)
	secretsClient := clientSet.CoreV1().Secrets(namespace)
	ctx := context.Background()

	repo := NewServiceCertificatesRepo(testutils.SensorOwnerReference(sensorDeployment)[0], namespace, secretsClient)
	persistedCertificates, err := repo.EnsureServiceCertificates(ctx, securedClusterCertificateSet)

	assert.NoError(t, err)
	protoassert.SlicesEqual(t, securedClusterCertificateSet.ServiceCerts, persistedCertificates)

	expectedSecretNames := []string{sensorSecretName, collectorSecretName, admissionControlSecretName,
		scannerSecretName, scannerDbSecretName,
		scannerV4IndexerSecretName, scannerV4DbSecretName}
	assert.Equal(t, len(expectedSecretNames), len(persistedCertificates))

	for _, secretName := range expectedSecretNames {
		_, err = secretsClient.Get(ctx, secretName, metav1.GetOptions{})
		assert.NoError(t, err)
	}
}
