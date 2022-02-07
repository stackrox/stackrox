package localscanner

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	appsApiv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	// ErrSensorDoesNotOwnCertSecrets indicates that this component won't be updating the certificates in
	// the secrets because the owner of the secrets is not the deployment for sensor.
	ErrSensorDoesNotOwnCertSecrets = errors.New("sensor deployment does not own certificate secrets")

	errForServiceFormat                         = "for service type %q"
	_                   serviceCertificatesRepo = (*serviceCertificatesRepoSecretsImpl)(nil)
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
type serviceCertificatesRepoSecretsImpl struct {
	secrets       map[storage.ServiceType]ServiceCertSecretSpec
	secretsClient corev1.SecretInterface
}

// ServiceCertSecretSpec species the name of the secret where certificates for a service are stored, and
// the secret data keys where each certificate file is stored.
type ServiceCertSecretSpec struct {
	secretName          string
	caCertFileName      string
	serviceCertFileName string
	serviceKeyFileName  string
}

// NewServiceCertificatesRepo creates a new serviceCertificatesRepoSecretsImpl that persists
// certificates for the specified services in k8s secrets with the secret name and secret data
// path specified in ServiceCertSecretSpec.
// Returns ErrSensorDoesNotOwnCertSecrets in case some secret doesn't have sensorDeployment as owner.
// If some secret does not exist then it creates it in same namespace as sensorDeployment, and with
// sensorDeployment as owner, populating the data of the new secrets with the corresponding certificates
// in initialCerts.
func NewServiceCertificatesRepo(ctx context.Context, scannerSpec, scannerDBSpec ServiceCertSecretSpec,
	sensorDeployment *appsApiv1.Deployment, initialCerts *storage.TypedServiceCertificateSet,
	secretsClient corev1.SecretInterface) (serviceCertificatesRepo, error) {
	repo := &serviceCertificatesRepoSecretsImpl{
		secrets: map[storage.ServiceType]ServiceCertSecretSpec{
			storage.ServiceType_SCANNER_SERVICE:    scannerSpec,
			storage.ServiceType_SCANNER_DB_SERVICE: scannerDBSpec,
		},
		secretsClient: secretsClient,
	}
	if err := repo.setupSecrets(ctx, sensorDeployment, initialCerts); err != nil {
		return nil, errors.Wrap(err, "setting up secrets")
	}

	return repo, nil
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
	for serviceType, secretSpec := range r.secrets {
		if err := r.getServiceCertificate(ctx, serviceType, secretSpec, certificates, &firstServiceTypeWithCA); err != nil {
			getErr = multierror.Append(getErr, err)
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

func (r *serviceCertificatesRepoSecretsImpl) getServiceCertificate(ctx context.Context,
	serviceType storage.ServiceType, secretSpec ServiceCertSecretSpec,
	certificates *storage.TypedServiceCertificateSet,
	firstServiceTypeWithCA *storage.ServiceType) error {
	retrievedSecret, err := r.secretsClient.Get(ctx, secretSpec.secretName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, errForServiceFormat, serviceType)
	}
	secretData := retrievedSecret.Data
	if secretData == nil {
		return errors.Wrapf(err, "missing for secret data for service type %q", serviceType)
	}
	if certificates.GetCaPem() == nil {
		certificates.CaPem = secretData[secretSpec.caCertFileName]
		*firstServiceTypeWithCA = serviceType
	} else {
		if !bytes.Equal(certificates.GetCaPem(), secretData[secretSpec.caCertFileName]) {
			return errors.Errorf("found different CA PEM in secret Data for service types %q and %q",
				firstServiceTypeWithCA, serviceType)
		}
	}
	certificates.ServiceCerts = append(certificates.ServiceCerts, &storage.TypedServiceCertificate{
		ServiceType: serviceType,
		Cert: &storage.ServiceCertificate{
			CertPem: secretData[secretSpec.serviceCertFileName],
			KeyPem:  secretData[secretSpec.serviceKeyFileName],
		},
	})

	return nil
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

func (r *serviceCertificatesRepoSecretsImpl) putServiceCertificate(ctx context.Context, caPem []byte,
	cert *storage.TypedServiceCertificate) error {

	secretSpec, ok := r.secrets[cert.GetServiceType()]
	if !ok {
		// we don't know how to persist this.
		return errors.Errorf("unkown service type %q", cert.GetServiceType())
	}
	secretData, err := r.secretDataForCertificate(caPem, cert)
	if err != nil {
		return errors.Wrapf(err, errForServiceFormat, cert.GetServiceType())
	}
	patch := []patchByteMap{{
		Op:    "replace",
		Path:  "/data",
		Value: secretData,
	}}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return errors.Wrapf(err, errForServiceFormat, cert.GetServiceType())
	}
	if _, err = r.secretsClient.Patch(ctx, secretSpec.secretName, k8sTypes.JSONPatchType, patchBytes,
		metav1.PatchOptions{}); err != nil {
		return errors.Wrapf(err, errForServiceFormat, cert.GetServiceType())
	}

	return nil
}

type patchByteMap struct {
	Op    string            `json:"op"`
	Path  string            `json:"path"`
	Value map[string][]byte `json:"value"`
}

// setupSecrets setups the k8s secrets where we store the certificates.
// - In case the secret doesn't have sensorDeployment as owner, this returns ErrSensorDoesNotOwnCertSecrets.
// - In case the secret doesn't exist this creates it setting sensorDeployment as owner, with cert stored
// 	 in the secret data.
func (r *serviceCertificatesRepoSecretsImpl) setupSecrets(ctx context.Context, sensorDeployment *appsApiv1.Deployment,
	initialCerts *storage.TypedServiceCertificateSet) error {
	for serviceType, secretSpec := range r.secrets {
		serviceCert, err := r.certificateForService(initialCerts, serviceType)
		if err != nil {
			return errors.Wrapf(err, errForServiceFormat, serviceType)
		}
		_, err = r.setupSecret(ctx, initialCerts.GetCaPem(), serviceCert, sensorDeployment, secretSpec.secretName)
		if err != nil {
			return errors.Wrapf(err, errForServiceFormat, serviceType)
		}
	}

	return nil
}

func (r *serviceCertificatesRepoSecretsImpl) setupSecret(ctx context.Context,
	caPem []byte, cert *storage.TypedServiceCertificate,
	sensorDeployment *appsApiv1.Deployment, secretName string) (*v1.Secret, error) {
	secret, err := r.secretsClient.Get(ctx, secretName, metav1.GetOptions{})

	if k8sErrors.IsNotFound(err) {
		secretData, err := r.secretDataForCertificate(caPem, cert)
		if err != nil {
			return nil, err
		}
		newSecret, createErr := r.secretsClient.Create(ctx, &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:            secretName,
				Namespace:       sensorDeployment.GetNamespace(),
				OwnerReferences: ownerReferenceFor(sensorDeployment),
			},
			Data: secretData,
		}, metav1.CreateOptions{})
		if createErr != nil {
			return nil, createErr
		}
		return newSecret, nil
	}

	if err != nil {
		return nil, err
	}

	ownerReferences := secret.GetOwnerReferences()
	if len(ownerReferences) != 1 {
		return nil, ErrSensorDoesNotOwnCertSecrets
	}

	ownerRef := ownerReferences[0]
	if ownerRef.UID != sensorDeployment.GetUID() {
		return nil, ErrSensorDoesNotOwnCertSecrets
	}

	return secret, nil
}

func ownerReferenceFor(sensorDeployment *appsApiv1.Deployment) []metav1.OwnerReference {
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(sensorDeployment, sensorDeployment.GroupVersionKind()),
	}
}

func (r *serviceCertificatesRepoSecretsImpl) certificateForService(certs *storage.TypedServiceCertificateSet,
	serviceType storage.ServiceType) (*storage.TypedServiceCertificate, error) {
	for _, cert := range certs.GetServiceCerts() {
		if cert.GetServiceType() == serviceType {
			return cert, nil
		}
	}
	return nil, errors.Errorf("no certificate found for service type %q", serviceType)
}

func (r *serviceCertificatesRepoSecretsImpl) secretDataForCertificate(caPem []byte,
	cert *storage.TypedServiceCertificate) (map[string][]byte, error) {
	secretSpec, ok := r.secrets[cert.GetServiceType()]
	if !ok {
		return nil, errors.Errorf("unkown service type %q", cert.GetServiceType())
	}
	return map[string][]byte{
		secretSpec.caCertFileName:      caPem,
		secretSpec.serviceCertFileName: cert.GetCert().GetCertPem(),
		secretSpec.serviceKeyFileName:  cert.GetCert().GetKeyPem(),
	}, nil
}
