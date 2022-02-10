package localscanner

import (
	"bytes"
	"context"
	"encoding/json"

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
	// ErrSensorDoesNotOwnCertSecrets indicates that this repository should not be updating the certificates in
	// the secrets because the owner of the secrets is not the deployment for sensor.
	ErrSensorDoesNotOwnCertSecrets = errors.New("sensor deployment does not own certificate secrets")

	// ErrDifferentCAForDifferentServiceTypes indicates that different service types have different values
	// for CA stored in their secrets.
	ErrDifferentCAForDifferentServiceTypes = errors.New("found different CA PEM in secret Data for different service types")

	// ErrMissingSecretData indicates some secret has no data.
	ErrMissingSecretData = errors.New("missing secret data")

	errForServiceFormat = "for service type %q"
)

// serviceCertificatesRepoSecretsImpl is a ServiceCertificatesRepo that uses k8s secrets for persistence.
type serviceCertificatesRepoSecretsImpl struct {
	secrets          map[storage.ServiceType]ServiceCertSecretSpec
	sensorDeployment *appsApiv1.Deployment
	secretsClient    corev1.SecretInterface
}

// ServiceCertSecretSpec species the name of the secret where certificates for a service are stored, and
// the secret data keys where each certificate file is stored.
type ServiceCertSecretSpec struct {
	secretName          string
	caCertFileName      string
	serviceCertFileName string
	serviceKeyFileName  string
}

// NewServiceCertificatesRepo creates a new serviceCertificatesRepoSecretsImpl that persists certificates for
// scanner and scanner DB in k8s secrets with the secret name and secret data path specified in ServiceCertSecretSpec.
// Returns ErrSensorDoesNotOwnCertSecrets in case some secret doesn't have sensorDeployment as owner.
// In case some secret does not exist then it creates it in same namespace as sensorDeployment, and with
// sensorDeployment as owner, populating the secret data with the corresponding certificates in initialCerts.
func NewServiceCertificatesRepo(ctx context.Context, scannerSpec, scannerDBSpec ServiceCertSecretSpec,
	sensorDeployment *appsApiv1.Deployment, initialCertsSupplier func(context.Context) (*storage.TypedServiceCertificateSet, error),
	secretsClient corev1.SecretInterface) (*serviceCertificatesRepoSecretsImpl, error) {
	repo := &serviceCertificatesRepoSecretsImpl{
		secrets: map[storage.ServiceType]ServiceCertSecretSpec{
			storage.ServiceType_SCANNER_SERVICE:    scannerSpec,
			storage.ServiceType_SCANNER_DB_SERVICE: scannerDBSpec,
		},
		sensorDeployment: sensorDeployment,
		secretsClient:    secretsClient,
	}
	if err := repo.setupSecrets(ctx, sensorDeployment, initialCertsSupplier); err != nil {
		return nil, errors.Wrap(err, "setting up secrets")
	}

	return repo, nil
}

// GetServiceCertificates behaves as follows in case of missing data in the secrets:
// - Fails with ErrMissingSecretData in case any secret has no data.
// - If the data for a secret is missing some expecting key then the corresponding field in the TypedServiceCertificate.
//   for that secret will contain a zero value.
// - Fails with ErrDifferentCAForDifferentServiceTypes in case the CA is not the same in all secrets.
// - Fails ErrSensorDoesNotOwnCertSecrets in case sensor deployment is not the sole owner of all secrets.
func (r *serviceCertificatesRepoSecretsImpl) GetServiceCertificates(ctx context.Context) (*storage.TypedServiceCertificateSet, error) {
	certificates := &storage.TypedServiceCertificateSet{}
	certificates.ServiceCerts = make([]*storage.TypedServiceCertificate, 0)
	for serviceType, secretSpec := range r.secrets {
		certificate, ca, err := r.getServiceCertificate(ctx, serviceType, secretSpec)
		if err != nil {
			return nil, err
		}
		if certificates.GetCaPem() == nil {
			certificates.CaPem = ca
		} else {
			if !bytes.Equal(certificates.GetCaPem(), ca) {
				return nil, ErrDifferentCAForDifferentServiceTypes
			}
		}
		certificates.ServiceCerts = append(certificates.ServiceCerts, certificate)
		// on context cancellation abort getting other secrets.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}

	return certificates, nil
}

func (r *serviceCertificatesRepoSecretsImpl) getServiceCertificate(ctx context.Context, serviceType storage.ServiceType,
	secretSpec ServiceCertSecretSpec) (cert *storage.TypedServiceCertificate, ca []byte, err error) {
	secret, getErr := r.secretsClient.Get(ctx, secretSpec.secretName, metav1.GetOptions{})
	if getErr != nil {
		return nil, nil, getErr
	}

	ownerReferences := secret.GetOwnerReferences()
	if len(ownerReferences) != 1 {
		return nil, nil, ErrSensorDoesNotOwnCertSecrets
	}

	ownerRef := ownerReferences[0]
	sensorDeploymentGVK := r.sensorDeployment.GroupVersionKind()
	if !(ownerRef.APIVersion == sensorDeploymentGVK.GroupVersion().String() &&
		ownerRef.Kind == sensorDeploymentGVK.Kind &&
		ownerRef.Name == r.sensorDeployment.GetName() &&
		ownerRef.UID == r.sensorDeployment.GetUID()) {
		return nil, nil, ErrSensorDoesNotOwnCertSecrets
	}

	secretData := secret.Data
	if secretData == nil {
		return nil, nil, ErrMissingSecretData
	}

	return &storage.TypedServiceCertificate{
		ServiceType: serviceType,
		Cert: &storage.ServiceCertificate{
			CertPem: secretData[secretSpec.serviceCertFileName],
			KeyPem:  secretData[secretSpec.serviceKeyFileName],
		},
	}, secretData[secretSpec.caCertFileName], nil
}

// PutServiceCertificates is idempotent but not atomic in sense that on error some secrets might be persisted
// while others are not.
// Edge cases:
// - Fails for certificates with a service type that doesn't appear in r.secrets, as we don't know where to store them.
// - Not all services types in r.secrets are required to appear in certificates, missing service types are just skipped.
func (r *serviceCertificatesRepoSecretsImpl) PutServiceCertificates(ctx context.Context,
	certificates *storage.TypedServiceCertificateSet) error {
	caPem := certificates.GetCaPem()
	for _, cert := range certificates.GetServiceCerts() {
		if err := r.putServiceCertificate(ctx, caPem, cert); err != nil {
			return err
		}

		// on context cancellation abort putting other secrets.
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return nil
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
	patch := []patchSecretDataByteMap{{
		Op:    "replace",
		Path:  "/data",
		Value: secretData,
	}}
	patchBytes, marshallingErr := json.Marshal(patch)
	if marshallingErr != nil {
		return errors.Wrapf(marshallingErr, errForServiceFormat, cert.GetServiceType())
	}
	if _, patchErr := r.secretsClient.Patch(ctx, secretSpec.secretName, k8sTypes.JSONPatchType, patchBytes,
		metav1.PatchOptions{}); patchErr != nil {
		return errors.Wrapf(patchErr, errForServiceFormat, cert.GetServiceType())
	}

	return nil
}

type patchSecretDataByteMap struct {
	Op    string            `json:"op"`
	Path  string            `json:"path"`
	Value map[string][]byte `json:"value"`
}

// setupSecrets setups the k8s secrets where we store the certificates.
// - In case a secret doesn't have sensorDeployment as owner, this returns ErrSensorDoesNotOwnCertSecrets.
// - In case a secret doesn't exist this creates it setting sensorDeployment as owner, with cert stored
// 	 in the secret data.
func (r *serviceCertificatesRepoSecretsImpl) setupSecrets(ctx context.Context, sensorDeployment *appsApiv1.Deployment,
	initialCertsSupplier func(context.Context) (*storage.TypedServiceCertificateSet, error)) error {
	for serviceType, secretSpec := range r.secrets {
		_, err := r.setupSecret(ctx, serviceType, initialCertsSupplier, sensorDeployment, secretSpec.secretName)
		if err != nil {
			return errors.Wrapf(err, errForServiceFormat, serviceType)
		}
	}

	return nil
}

func (r *serviceCertificatesRepoSecretsImpl) setupSecret(ctx context.Context,
	serviceType storage.ServiceType,
	initialCertsSupplier func(context.Context) (*storage.TypedServiceCertificateSet, error),
	sensorDeployment *appsApiv1.Deployment, secretName string) (*v1.Secret, error) {
	secret, err := r.secretsClient.Get(ctx, secretName, metav1.GetOptions{})

	if k8sErrors.IsNotFound(err) {
		initialCerts, getInitialCertsErr := initialCertsSupplier(ctx)
		if getInitialCertsErr != nil {
			return nil, getInitialCertsErr
		}
		cert, getCertErr := r.certificateForService(initialCerts, serviceType)
		if getCertErr != nil {
			return nil, getCertErr
		}
		secretData, dataForCertErr := r.secretDataForCertificate(initialCerts.GetCaPem(), cert)
		if dataForCertErr != nil {
			return nil, dataForCertErr
		}
		sensorDeploymentGVK := sensorDeployment.GroupVersionKind()
		blockOwnerDeletion := false
		isController := false
		newSecret, createErr := r.secretsClient.Create(ctx, &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: sensorDeployment.GetNamespace(),
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         sensorDeploymentGVK.GroupVersion().String(),
						Kind:               sensorDeploymentGVK.Kind,
						Name:               sensorDeployment.GetName(),
						UID:                sensorDeployment.GetUID(),
						BlockOwnerDeletion: &blockOwnerDeletion,
						Controller:         &isController,
					},
				},
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
