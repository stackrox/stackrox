package localscanner

import (
	"bytes"
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mtls"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	// ErrSensorDoesNotOwnCertSecrets indicates that this repository should not be updating the certificates in
	// the secrets because the owner of the secrets is not the deployment for sensor.
	ErrSensorDoesNotOwnCertSecrets = errors.New("sensor deployment does not own certificate secrets")

	errForServiceFormat                         = "for service type %q"
	_                   ServiceCertificatesRepo = (*serviceCertificatesRepoSecretsImpl)(nil)
)

// ServiceCertificatesRepo is in charge of persisting and retrieving a set of service certificates, thus implementing
// the [repository pattern](https://martinfowler.com/eaaCatalog/repository.html) for *storage.TypedServiceCertificateSet.
type ServiceCertificatesRepo interface {
	// GetServiceCertificates retrieves the certificates from permanent storage.
	GetServiceCertificates(ctx context.Context) (*storage.TypedServiceCertificateSet, error)
	// PutServiceCertificates persists the certificates on permanent storage.
	PutServiceCertificates(ctx context.Context, certificates *storage.TypedServiceCertificateSet) error
}

// serviceCertificatesRepoSecretsImpl is a ServiceCertificatesRepo that uses k8s secrets for persistence.
type serviceCertificatesRepoSecretsImpl struct {
	secrets       map[storage.ServiceType]ServiceCertSecretSpec
	secretsClient corev1.SecretInterface
	ownerReference metav1.OwnerReference
	namespace      string
}

// ServiceCertSecretSpec species the name of the secret where certificates for a service are stored, and
// the secret data keys where each certificate file is stored.
type ServiceCertSecretSpec struct {
	secretName          string
	caCertFileName      string
	serviceCertFileName string
	serviceKeyFileName  string
	configureSecretFunc func(certificate storage.TypedServiceCertificate) v1.Secret
}

// NewServiceCertificatesRepo creates a new serviceCertificatesRepoSecretsImpl that persists certificates for
// scanner and scanner DB in k8s secrets with the secret name and secret data path specified in ServiceCertSecretSpec.
// Returns ErrSensorDoesNotOwnCertSecrets in case some secret doesn't have sensorDeployment as owner.
// In case some secret does not exist then it creates it in same namespace as sensorDeployment, and with
// sensorDeployment as owner, populating the secret data with the corresponding certificates in initialCerts.
func NewServiceCertificatesRepo(ownerReference metav1.OwnerReference, namespace string, secretsClient corev1.SecretInterface) (ServiceCertificatesRepo, error) {
	repo := &serviceCertificatesRepoSecretsImpl{
		secrets: map[storage.ServiceType]ServiceCertSecretSpec{
			storage.ServiceType_SCANNER_SERVICE: ServiceCertSecretSpec{
				secretName: "scanner-slim-tls",
				caCertFileName: mtls.CACertFileName,
				serviceCertFileName: mtls.ServiceCertFileName,
				serviceKeyFileName: mtls.ServiceKeyFileName,
			},
			storage.ServiceType_SCANNER_DB_SERVICE: ServiceCertSecretSpec{
				secretName: "scanner-db-slim-tls",
				caCertFileName: mtls.CACertFileName,
				serviceCertFileName: mtls.ServiceCertFileName,
				serviceKeyFileName: mtls.ServiceKeyFileName,
			},
		},
		secretsClient: secretsClient,
		ownerReference: ownerReference,
		namespace: namespace,
	}

	return repo, nil
}

// GetServiceCertificates behaves as follows in case of missing data in the secrets:
// - if a secret has no data then the certificates won't contain a TypedServiceCertificate for the corresponding
//   service type.
// - if the data for a secret is missing some expecting key then the corresponding field in the TypedServiceCertificate
//   for that secret will contain a zero value.
func (r *serviceCertificatesRepoSecretsImpl) GetServiceCertificates(ctx context.Context) (*storage.TypedServiceCertificateSet, error) {
	certificates := &storage.TypedServiceCertificateSet{}
	certificates.ServiceCerts = make([]*storage.TypedServiceCertificate, 0)
	var getErr error
	for serviceType, _ := range r.secrets {
		typedCertsSet, err := r.getServiceCertificate(ctx, serviceType)
		if err != nil {
			getErr = multierror.Append(err)
		}

		certificates.ServiceCerts = append(certificates.ServiceCerts, typedCertsSet.ServiceCerts...)
		if certificates.CaPem == nil {
			certificates.CaPem = typedCertsSet.CaPem
		}
		if !bytes.Equal(certificates.CaPem, typedCertsSet.CaPem) {
			return nil, errors.Errorf("CA cert for service %q does not match", serviceType)
		}

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

func (r *serviceCertificatesRepoSecretsImpl) getServiceCertificate(ctx context.Context, serviceType storage.ServiceType) (*storage.TypedServiceCertificateSet, error) {
	secretInfo, ok := r.secrets[serviceType]
	if !ok {
		return nil, errors.Errorf("Secret repo does not support services type %q", serviceType)
	}

	secret, err := r.secretsClient.Get(ctx, secretInfo.secretName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, errForServiceFormat, serviceType)
	}

	return &storage.TypedServiceCertificateSet{
		ServiceCerts: []*storage.TypedServiceCertificate{{
			ServiceType: serviceType,
			Cert: &storage.ServiceCertificate{
				CertPem: secret.Data[secretInfo.serviceCertFileName],
				KeyPem:  secret.Data[secretInfo.serviceKeyFileName],
			},
		}},
		CaPem: secret.Data[secretInfo.caCertFileName],
	}, nil
}

// PutServiceCertificates is idempotent but not atomic in sense that on error some secrets might be persisted
// while others are not.
// Edge cases:
// - Fails for certificates with a service type that doesn't appear in r.secrets, as we don't know where to store them.
// - Not all services types in r.secrets are required to appear in certificates, missing service types are just skipped.
func (r *serviceCertificatesRepoSecretsImpl) PutServiceCertificates(ctx context.Context,
	certificates *storage.TypedServiceCertificateSet) error {
	var putErr error
	caPem := certificates.GetCaPem()
	for _, cert := range certificates.GetServiceCerts() {
		if err := r.putServiceCertificate(ctx, caPem, cert); err != nil {
			putErr = multierror.Append(putErr, err)
		}

		// on context cancellation abort putting other secrets.
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return putErr
}

func (r *serviceCertificatesRepoSecretsImpl) putServiceCertificate(ctx context.Context, caPem []byte, cert *storage.TypedServiceCertificate) error {
	secretInfo, ok := r.secrets[cert.GetServiceType()]
	if !ok {
		// we don't know how to persist this.
		return errors.Errorf("unkown service type %q", cert.GetServiceType())
	}

	_, err := r.secretsClient.Update(ctx, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretInfo.secretName,
			Namespace: "",
		},
		Data: r.secretDataForCertificate(secretInfo, caPem, cert),
	}, metav1.UpdateOptions{})
	return err
}

// createSecrets setups the k8s secrets where we store the certificates.
// - In case a secret doesn't have sensorDeployment as owner, this returns ErrSensorDoesNotOwnCertSecrets.
// - In case a secret doesn't exist this creates it setting sensorDeployment as owner, with cert stored
// 	 in the secret data.
func (r *serviceCertificatesRepoSecretsImpl) createSecrets(ctx context.Context, typedCerts *storage.TypedServiceCertificateSet) error {
	for _, cert := range typedCerts.GetServiceCerts() {
		if _, ok := r.secrets[cert.GetServiceType()]; !ok {
			return errors.Errorf("Unknown service type %q", cert.GetServiceType())
		}

		_, err := r.createSecret(ctx, typedCerts.GetCaPem(), cert)
		if err != nil {
			// TODO(do-not-merge): handle multierror
			return err
		}
	}

	return nil
}

func (r *serviceCertificatesRepoSecretsImpl) createSecret(ctx context.Context, caPem []byte, typedCerts *storage.TypedServiceCertificate) (*v1.Secret, error) {
	secretInfo, ok := r.secrets[typedCerts.GetServiceType()]
	if !ok {
		return nil, errors.Errorf("not supported")
	}
	secret, err := r.secretsClient.Get(ctx, secretInfo.secretName, metav1.GetOptions{})

	if k8sErrors.IsNotFound(err) {
		newSecret, createErr := r.secretsClient.Create(ctx, &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretInfo.secretName,
				Namespace: r.namespace,
				OwnerReferences: []metav1.OwnerReference{r.ownerReference},
			},
			Data: r.secretDataForCertificate(secretInfo, caPem, typedCerts),
		}, metav1.CreateOptions{})
		if createErr != nil {
			return nil, createErr
		}
		return newSecret, nil
	}

	if err != nil {
		return nil, err
	}

	return secret, nil
}


func (r *serviceCertificatesRepoSecretsImpl) secretDataForCertificate(secretInfo ServiceCertSecretSpec, caPem []byte, cert *storage.TypedServiceCertificate) map[string][]byte {
	return map[string][]byte{
		secretInfo.caCertFileName:      caPem,
		secretInfo.serviceCertFileName: cert.GetCert().GetCertPem(),
		secretInfo.serviceKeyFileName:  cert.GetCert().GetKeyPem(),
	}
}
