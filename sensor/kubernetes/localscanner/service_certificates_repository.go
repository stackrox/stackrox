package localscanner

import (
	"bytes"
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

// serviceCertificatesRepo is in charge of persisting and retrieving a set of service certificates, thus implementing
// the [repository pattern](https://martinfowler.com/eaaCatalog/repository.html) for *storage.TypedServiceCertificateSet.
type serviceCertificatesRepo interface {
	// getServiceCertificates retrieves the certificates from permanent storage.
	getServiceCertificates(ctx context.Context) (*storage.TypedServiceCertificateSet, error)
	// putServiceCertificates persists the certificates on permanent storage.
	putServiceCertificates(ctx context.Context, certificates *storage.TypedServiceCertificateSet) error
}

// serviceCertificatesRepoSecretsImpl is a serviceCertificatesRepo that uses k8s secrets for persistence.
// All fields except Data are respected when persisting the secrets.
// Invariants:
// - secrets and secretsClient are read-only, except the field Data of the entries in secrets.
// - No secret in secrets is nil.
type serviceCertificatesRepoSecretsImpl struct {
	secrets       map[storage.ServiceType]*v1.Secret
	secretsClient corev1.SecretInterface
}

// newServiceCertificatesRepo creates a new serviceCertificatesRepoSecretsImpl that persists
// certificates for the specified services in the corresponding k8s secrets.
func newServiceCertificatesRepo(secrets map[storage.ServiceType]*v1.Secret,
	secretsClient corev1.SecretInterface) (serviceCertificatesRepo, error) {
	for serviceType, secret := range secrets {
		if secret == nil {
			return nil, errors.Errorf("nil secrets for service type %q", serviceType)
		}
	}
	return &serviceCertificatesRepoSecretsImpl{
		secrets:       secrets,
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
	var firstServiceTypeWithCA storage.ServiceType
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
			certificates.CaPem = secretData[mtls.CACertFileName]
			firstServiceTypeWithCA = serviceType
		} else {
			if !bytes.Equal(certificates.GetCaPem(), secretData[mtls.CACertFileName]) {
				return nil, errors.Errorf("found different CA PEM in secret Data for service types %q and %q",
					firstServiceTypeWithCA, serviceType)
			}
		}
		certificates.ServiceCerts = append(certificates.ServiceCerts, &storage.TypedServiceCertificate{
			ServiceType: serviceType,
			Cert: &storage.ServiceCertificate{
				CertPem: secretData[mtls.ServiceCertFileName],
				KeyPem:  secretData[mtls.ServiceKeyFileName],
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

// putServiceCertificates is idempotent but not atomic in sense that on error some secrets might be persisted
// while others are not.
// Edge cases:
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
