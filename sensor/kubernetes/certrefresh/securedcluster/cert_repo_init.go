package securedcluster

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/securedcluster"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certrepo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// NewServiceCertificatesRepo creates a new ServiceCertificatesRepoSecrets that persists certificates for
// all the Secured Cluster services in k8s secrets.
// Those secrets have to have ownerReference as the only owner reference.
func NewServiceCertificatesRepo(ownerReference metav1.OwnerReference, namespace string,
	secretsClient corev1.SecretInterface) certrepo.ServiceCertificatesRepo {

	secretsByServiceType := map[storage.ServiceType]certrepo.ServiceCertSecretSpec{
		storage.ServiceType_SENSOR_SERVICE:             certrepo.NewServiceCertSecretSpec(securedcluster.SensorTLSSecretName),
		storage.ServiceType_COLLECTOR_SERVICE:          certrepo.NewServiceCertSecretSpec(securedcluster.CollectorTLSSecretName),
		storage.ServiceType_ADMISSION_CONTROL_SERVICE:  certrepo.NewServiceCertSecretSpec(securedcluster.AdmissionControlTLSSecretName),
		storage.ServiceType_SCANNER_SERVICE:            certrepo.NewServiceCertSecretSpec(securedcluster.ScannerTLSSecretName),
		storage.ServiceType_SCANNER_DB_SERVICE:         certrepo.NewServiceCertSecretSpec(securedcluster.ScannerDbTLSSecretName),
		storage.ServiceType_SCANNER_V4_INDEXER_SERVICE: certrepo.NewServiceCertSecretSpec(securedcluster.ScannerV4IndexerTLSSecretName),
		storage.ServiceType_SCANNER_V4_DB_SERVICE:      certrepo.NewServiceCertSecretSpec(securedcluster.ScannerV4DbTLSSecretName),
	}

	return &certrepo.ServiceCertificatesRepoSecrets{
		Secrets:        secretsByServiceType,
		OwnerReference: ownerReference,
		Namespace:      namespace,
		SecretsClient:  secretsClient,
	}
}
