package localscanner

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
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
	scannerServiceType         = storage.ServiceType_SCANNER_SERVICE
	serviceCertificate         = createServiceCertificate(scannerServiceType)
	emptyPersistedCertificates = make([]*storage.TypedServiceCertificate, 0)
	certificateSet             = &storage.TypedServiceCertificateSet{
		CaPem: make([]byte, 2),
		ServiceCerts: []*storage.TypedServiceCertificate{
			serviceCertificate,
		},
	}
	scannersCertificateSet = &storage.TypedServiceCertificateSet{
		CaPem: make([]byte, 2),
		ServiceCerts: []*storage.TypedServiceCertificate{
			createServiceCertificate(storage.ServiceType_SCANNER_SERVICE),
			createServiceCertificate(storage.ServiceType_SCANNER_DB_SERVICE),
			createServiceCertificate(storage.ServiceType_SCANNER_V4_INDEXER_SERVICE),
			createServiceCertificate(storage.ServiceType_SCANNER_V4_DB_SERVICE),
		},
	}
)

func TestLocalScannerCertificateRepo(t *testing.T) {
	suite.Run(t, new(localScannerCertificateRepoSuite))
}

type localScannerCertificateRepoSuite struct {
	suite.Suite
}

func (s *localScannerCertificateRepoSuite) TestCreateSecretsNoCertificatesSuccess() {
	clientSet := fake.NewSimpleClientset(sensorDeployment)
	secretsClient := clientSet.CoreV1().Secrets(namespace)
	repo := NewServiceCertificatesRepo(sensorOwnerReference()[0], namespace, secretsClient)

	persistedCertificates, err := repo.EnsureServiceCertificates(context.Background(), nil)
	protoassert.SlicesEqual(s.T(), emptyPersistedCertificates, persistedCertificates)
	s.NoError(err)
}

func (s *localScannerCertificateRepoSuite) TestEnsureServiceCertificateMissingSecretSuccess() {
	clientSet := fake.NewSimpleClientset(sensorDeployment)
	secretsClient := clientSet.CoreV1().Secrets(namespace)
	repo := NewServiceCertificatesRepo(sensorOwnerReference()[0], namespace, secretsClient)

	persistedCertificates, err := repo.EnsureServiceCertificates(context.Background(), certificateSet)

	protoassert.SlicesEqual(s.T(), certificateSet.ServiceCerts, persistedCertificates)
	s.NoError(err)
}

func (s *localScannerCertificateRepoSuite) TestEnsureServiceCertificatesForScannerV4() {
	testutils.MustUpdateFeature(s.T(), features.ScannerV4, true)
	clientSet := fake.NewSimpleClientset(sensorDeployment)
	secretsClient := clientSet.CoreV1().Secrets(namespace)
	ctx := context.Background()
	repo := NewServiceCertificatesRepo(sensorOwnerReference()[0], namespace, secretsClient)

	persistedCertificates, err := repo.EnsureServiceCertificates(ctx, scannersCertificateSet)
	s.NoError(err)
	protoassert.SlicesEqual(s.T(), scannersCertificateSet.ServiceCerts, persistedCertificates)
	expectedSecretNames := []string{scannerSecretName, scannerDbSecretName,
		scannerV4IndexerSecretName, scannerV4DbSecretName}
	for _, secretName := range expectedSecretNames {
		_, err = secretsClient.Get(ctx, secretName, metav1.GetOptions{})
		s.NoError(err)
	}
}

func (s *localScannerCertificateRepoSuite) TestEnsureCertificatesScannerV4IgnoredWhenDisabled() {
	testutils.MustUpdateFeature(s.T(), features.ScannerV4, false)
	clientSet := fake.NewSimpleClientset(sensorDeployment)
	secretsClient := clientSet.CoreV1().Secrets(namespace)
	ctx := context.Background()
	repo := NewServiceCertificatesRepo(sensorOwnerReference()[0], namespace, secretsClient)
	scannerV2Certificates := []*storage.TypedServiceCertificate{
		createServiceCertificate(storage.ServiceType_SCANNER_SERVICE),
		createServiceCertificate(storage.ServiceType_SCANNER_DB_SERVICE),
	}

	persistedCertificates, err := repo.EnsureServiceCertificates(context.Background(), scannersCertificateSet)
	s.NoError(err)
	protoassert.SlicesEqual(s.T(), scannerV2Certificates, persistedCertificates)
	_, err = secretsClient.Get(ctx, scannerSecretName, metav1.GetOptions{})
	s.NoError(err)
	_, err = secretsClient.Get(ctx, scannerDbSecretName, metav1.GetOptions{})
	s.NoError(err)
	_, err = secretsClient.Get(ctx, scannerV4IndexerSecretName, metav1.GetOptions{})
	s.ErrorContains(err, "not found")
	_, err = secretsClient.Get(ctx, scannerV4DbSecretName, metav1.GetOptions{})
	s.ErrorContains(err, "not found")
}

func sensorOwnerReference() []metav1.OwnerReference {
	sensorDeploymentGVK := sensorDeployment.GroupVersionKind()
	blockOwnerDeletion := false
	isController := false
	return []metav1.OwnerReference{
		{
			APIVersion:         sensorDeploymentGVK.GroupVersion().String(),
			Kind:               sensorDeploymentGVK.Kind,
			Name:               sensorDeployment.GetName(),
			UID:                sensorDeployment.GetUID(),
			BlockOwnerDeletion: &blockOwnerDeletion,
			Controller:         &isController,
		},
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
