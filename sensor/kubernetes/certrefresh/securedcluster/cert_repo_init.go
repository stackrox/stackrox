package securedcluster

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certrepo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	// These secret names follow a different convention than our legacy secrets (e.g. sensor-tls, scanner-tls etc.),
	// so that they can both exist in parallel. This is in order to not create conflicts with automations of existing
	// deployments that might provide those legacy secrets.
	sensorSecretName           = "tls-cert-sensor"             // #nosec G101 not a hardcoded credential
	collectorSecretName        = "tls-cert-collector"          // #nosec G101 not a hardcoded credential
	admissionControlSecretName = "tls-cert-admission-control"  // #nosec G101 not a hardcoded credential
	scannerSecretName          = "tls-cert-scanner"            // #nosec G101 not a hardcoded credential
	scannerDbSecretName        = "tls-cert-scanner-db"         // #nosec G101 not a hardcoded credential
	scannerV4IndexerSecretName = "tls-cert-scanner-v4-indexer" // #nosec G101 not a hardcoded credential
	scannerV4DbSecretName      = "tls-cert-scanner-v4-db"      // #nosec G101 not a hardcoded credential
)

// NewServiceCertificatesRepo creates a new ServiceCertificatesRepoSecrets that persists certificates for
// all the Secured Cluster services in k8s secrets.
// Those secrets have to have ownerReference as the only owner reference.
func NewServiceCertificatesRepo(ownerReference metav1.OwnerReference, namespace string,
	secretsClient corev1.SecretInterface) certrepo.ServiceCertificatesRepo {

	secretsByServiceType := map[storage.ServiceType]certrepo.ServiceCertSecretSpec{
		storage.ServiceType_SENSOR_SERVICE:             certrepo.NewServiceCertSecretSpec(sensorSecretName),
		storage.ServiceType_COLLECTOR_SERVICE:          certrepo.NewServiceCertSecretSpec(collectorSecretName),
		storage.ServiceType_ADMISSION_CONTROL_SERVICE:  certrepo.NewServiceCertSecretSpec(admissionControlSecretName),
		storage.ServiceType_SCANNER_SERVICE:            certrepo.NewServiceCertSecretSpec(scannerSecretName),
		storage.ServiceType_SCANNER_DB_SERVICE:         certrepo.NewServiceCertSecretSpec(scannerDbSecretName),
		storage.ServiceType_SCANNER_V4_INDEXER_SERVICE: certrepo.NewServiceCertSecretSpec(scannerV4IndexerSecretName),
		storage.ServiceType_SCANNER_V4_DB_SERVICE:      certrepo.NewServiceCertSecretSpec(scannerV4DbSecretName),
	}

	return &certrepo.ServiceCertificatesRepoSecrets{
		Secrets:        secretsByServiceType,
		OwnerReference: ownerReference,
		Namespace:      namespace,
		SecretsClient:  secretsClient,
	}
}
