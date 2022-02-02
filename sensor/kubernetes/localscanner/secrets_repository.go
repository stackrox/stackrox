package localscanner

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mtls"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	_ serviceCertificatesRepo = (*serviceCertificatesRepoSecretsImpl)(nil)
)

// serviceCertificatesRepo is in charge of persisting and retrieving a set of secrets corresponding to a fixed
// set of service types into k8s, thus implementing the
// [repository pattern](https://martinfowler.com/eaaCatalog/repository.html) for a map from service types
// to secrets and using the k8s API as persistence.
type serviceCertificatesRepo interface {
	// getSecrets retrieves the secrets from permanent storage.
	// getSecrets(ctx context.Context) (map[storage.ServiceType]*v1.Secret, error) // FIXME update comments and delete
	getServiceCertificates(ctx context.Context) (*storage.TypedServiceCertificateSet, error)
	// putSecrets persists the secrets on permanent storage.
	// - Returns an error in case some service in `secret` is not in the set of service types handled by the repository.
	// - `secrets` may miss an entry for some service type handled by the repository, in that case this only updates
	//   the secrets for the service types in `secrets`.
	// - This operation is idempotent but not atomic in sense that on error some of the secrets might be persisted
	//   while others are not.
	// putSecrets(ctx context.Context, secrets map[storage.ServiceType]*v1.Secret) error // FIXME update comments and delete
	putServiceCertificates(ctx context.Context, certificates *storage.TypedServiceCertificateSet) error
}

// serviceCertificatesRepoSecretsImpl is a serviceCertificatesRepo that uses k8s secrets for persistence.
// Invariants:
// - secrets and secretsClient are read-only, but the elements of secrets are read-write.
// - All secrets store the same CA PEM.
// - No secret in secrets is nil.
type serviceCertificatesRepoSecretsImpl struct {
	secrets     map[storage.ServiceType]*v1.Secret
	secretsClient corev1.SecretInterface
}

// newServiceCertificatesRepoWithSecretsPersistence creates a new serviceCertificatesRepo that handles secrets with the specified names and
// for the specified service types, and uses the k8s API for persistence.
func newServiceCertificatesRepoWithSecretsPersistence(secrets map[storage.ServiceType]*v1.Secret,
	secretsClient corev1.SecretInterface) (serviceCertificatesRepo, error) {
	for serviceType, secret := range secrets {
		if secret == nil {
			return nil, errors.Errorf("nil secrets for service type %q", serviceType)
		}
	}
	return &serviceCertificatesRepoSecretsImpl{
		secrets: secrets,
		secretsClient: secretsClient,
	}, nil
}

// getServiceCertificates behaves as follows in case of missing data in the secrets:
// - if a secret has no data then the certificates won't contain a TypedServiceCertificate for the corresponding
//   service type.
// - if the data for a secret is missing some expecting key then the corresponding field in the TypedServiceCertificate
//   for that secret will contain a zero value.
func (r *serviceCertificatesRepoSecretsImpl) getServiceCertificates(ctx context.Context) (*storage.TypedServiceCertificateSet, error) {
	certificates := &storage.TypedServiceCertificateSet{}
	certificates.ServiceCerts = make([]*storage.TypedServiceCertificate, 0)
	var getErr error
	for serviceType, secret := range r.secrets {
		// Invariant: no secret in r.secrets is nil.
		retrievedSecret, err := r.secretsClient.Get(ctx, secret.Name, metav1.GetOptions{})
		if err != nil {
			getErr = multierror.Append(getErr, errors.Wrapf(err, "for secret %s", secret.Name))
			continue
		}
		secretData := retrievedSecret.Data
		if secretData == nil {
			continue
		}
		if certificates.GetCaPem() == nil {
			// all secrets store the same CA PEM.
			certificates.CaPem = secretData[mtls.CACertFileName]
		}
		certificates.ServiceCerts = append(certificates.ServiceCerts, &storage.TypedServiceCertificate{
			ServiceType: serviceType,
			Cert: &storage.ServiceCertificate{
				CertPem: secretData[mtls.ServiceCertFileName],
				KeyPem: secretData[mtls.ServiceKeyFileName],
			},
		})

		// on context cancellation abort getting other secrets.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	if getErr != nil {
		return nil, getErr
	}

	return certificates, nil
}

// putServiceCertificates edge cases:
// - Fails for certificates with a service type that doesn't appear in r.secrets, as we don't know where to store them.
// - Not all services types in r.secrets are required to appear in certificates, missing service types are just skipped.
func (r *serviceCertificatesRepoSecretsImpl) putServiceCertificates(ctx context.Context, certificates *storage.TypedServiceCertificateSet) error {
	var putErr error
	caPem := certificates.GetCaPem()
	for _, cert := range certificates.GetServiceCerts() {
		secret, ok := r.secrets[cert.GetServiceType()]
		if !ok {
			// we don't know where to persist this.
			putErr = multierror.Append(putErr, errors.Errorf("unkown service type %s", cert.GetServiceType()))
			continue
		}
		// Invariant: no secret in r.secrets is nil.
		secret.Data = map[string][]byte{
			mtls.CACertFileName:      caPem,
			mtls.ServiceCertFileName: cert.GetCert().GetCertPem(),
			mtls.ServiceKeyFileName:  cert.GetCert().GetKeyPem(),
		}
		_, err := r.secretsClient.Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			putErr = multierror.Append(putErr, errors.Wrapf(err, "for secret %s", secret.Name))
		}

		// on context cancellation abort putting other secrets.
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return putErr
}