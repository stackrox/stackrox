package localscanner

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certrepo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	scannerSecretName          = "scanner-tls"            // #nosec G101 not a hardcoded credential
	scannerDbSecretName        = "scanner-db-tls"         // #nosec G101 not a hardcoded credential
	scannerV4IndexerSecretName = "scanner-v4-indexer-tls" // #nosec G101 not a hardcoded credential
	scannerV4DbSecretName      = "scanner-v4-db-tls"      // #nosec G101 not a hardcoded credential
)

// NewServiceCertificatesRepo creates a new ServiceCertificatesRepoSecrets that persists certificates for
// scannerV2, scanner DB, scannerV4 indexer and ScannerV4 DB in k8s secrets.
// Those secrets have to have ownerReference as the only owner reference.
func NewServiceCertificatesRepo(ownerReference metav1.OwnerReference, namespace string,
	secretsClient corev1.SecretInterface) certrepo.ServiceCertificatesRepo {

	secretsByServiceType := map[storage.ServiceType]certrepo.ServiceCertSecretSpec{
		storage.ServiceType_SCANNER_SERVICE:    certrepo.NewServiceCertSecretSpec(scannerSecretName),
		storage.ServiceType_SCANNER_DB_SERVICE: certrepo.NewServiceCertSecretSpec(scannerDbSecretName),
	}

	if features.ScannerV4.Enabled() {
		secretsByServiceType[storage.ServiceType_SCANNER_V4_INDEXER_SERVICE] = certrepo.NewServiceCertSecretSpec(scannerV4IndexerSecretName)
		secretsByServiceType[storage.ServiceType_SCANNER_V4_DB_SERVICE] = certrepo.NewServiceCertSecretSpec(scannerV4DbSecretName)
	}

	return &certrepo.ServiceCertificatesRepoSecrets{
		Secrets:        secretsByServiceType,
		OwnerReference: ownerReference,
		Namespace:      namespace,
		SecretsClient:  secretsClient,
	}
}
