package localscanner

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mtls"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	// ErrUnexpectedSecretsOwner indicates that this repository should not be updating the certificates in
	// the secrets they do not have the expected owner.
	ErrUnexpectedSecretsOwner = errors.New("unexpected owner for certificate secrets")

	// ErrDifferentCAForDifferentServiceTypes indicates that different service types have different values
	// for CA stored in their secrets.
	ErrDifferentCAForDifferentServiceTypes = errors.New("found different CA PEM in secret Data for different service types")

	// ErrMissingSecretData indicates some secret has no data.
	ErrMissingSecretData = errors.New("missing secret data")

	errForServiceFormat = "for service type %q"
)

// serviceCertificatesRepoSecretsImpl is a ServiceCertificatesRepo that uses k8s secrets for persistence.
type serviceCertificatesRepoSecretsImpl struct {
	secrets        map[storage.ServiceType]serviceCertSecretSpec
	ownerReference metav1.OwnerReference
	namespace      string
	secretsClient  corev1.SecretInterface
}

// serviceCertSecretSpec species the name of the secret where certificates for a service are stored, and
// the secret data keys where each certificate file is stored.
type serviceCertSecretSpec struct {
	secretName          string
	caCertFileName      string
	serviceCertFileName string
	serviceKeyFileName  string
}

// newServiceCertificatesRepo creates a new serviceCertificatesRepoSecretsImpl that persists certificates for
// scanner and scanner DB in k8s secrets that are expected to have ownerReference as the only owner reference.
func newServiceCertificatesRepo(ownerReference metav1.OwnerReference, namespace string,
	secretsClient corev1.SecretInterface) *serviceCertificatesRepoSecretsImpl {

	return &serviceCertificatesRepoSecretsImpl{
		secrets: map[storage.ServiceType]serviceCertSecretSpec{
			storage.ServiceType_SCANNER_SERVICE: {
				secretName:          "scanner-slim-tls",
				caCertFileName:      mtls.CACertFileName,
				serviceCertFileName: mtls.ServiceCertFileName,
				serviceKeyFileName:  mtls.ServiceKeyFileName,
			},
			storage.ServiceType_SCANNER_DB_SERVICE: {
				secretName:          "scanner-db-slim-tls",
				caCertFileName:      mtls.CACertFileName,
				serviceCertFileName: mtls.ServiceCertFileName,
				serviceKeyFileName:  mtls.ServiceKeyFileName,
			},
		},
		ownerReference: ownerReference,
		namespace:      namespace,
		secretsClient:  secretsClient,
	}
}

// getServiceCertificates behaves as follows in case of errors:
// - Fails as soon as the context is cancelled.
// - Otherwise, fails with ErrUnexpectedSecretsOwner in case the owner specified in the constructor is not the sole owner of all secrets.
// - Otherwise, fails with ErrMissingSecretData in case any secret has no data.
// - Otherwise, fails with ErrDifferentCAForDifferentServiceTypes in case the CA is not the same in all secrets.
// - Otherwise, fails with any k8s API error.
// - If the data for a secret is missing some expecting key then the corresponding field in the TypedServiceCertificate.
//   for that secret will contain a zero value.
func (r *serviceCertificatesRepoSecretsImpl) getServiceCertificates(ctx context.Context) (*storage.TypedServiceCertificateSet, error) {
	certificates := &storage.TypedServiceCertificateSet{}
	certificates.ServiceCerts = make([]*storage.TypedServiceCertificate, 0)

	errors := make(map[error]bool)
	for serviceType, secretSpec := range r.secrets {
		certificate, ca, err := r.getServiceCertificate(ctx, serviceType, secretSpec)
		if err != nil {
			errors[err] = true
			continue
		}
		if certificates.GetCaPem() == nil {
			certificates.CaPem = ca
		} else {
			if !bytes.Equal(certificates.GetCaPem(), ca) {
				errors[ErrDifferentCAForDifferentServiceTypes] = true
				continue
			}
		}
		certificates.ServiceCerts = append(certificates.ServiceCerts, certificate)
		// on context cancellation abort getting other secrets.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}

	// TODO: if this works abstract as "highestPriorityError" func
	if _, ok := errors[ErrUnexpectedSecretsOwner]; ok {
		return nil, ErrUnexpectedSecretsOwner
	}
	if _, ok := errors[ErrMissingSecretData]; ok {
		return nil, ErrMissingSecretData
	}
	if _, ok := errors[ErrDifferentCAForDifferentServiceTypes]; ok {
		return nil, ErrDifferentCAForDifferentServiceTypes
	}
	for err := range errors {
		return nil, err
	}

	return certificates, nil
}

func (r *serviceCertificatesRepoSecretsImpl) getServiceCertificate(ctx context.Context, serviceType storage.ServiceType,
	secretSpec serviceCertSecretSpec) (cert *storage.TypedServiceCertificate, ca []byte, err error) {
	secret, getErr := r.secretsClient.Get(ctx, secretSpec.secretName, metav1.GetOptions{})
	if getErr != nil {
		return nil, nil, getErr
	}

	ownerReferences := secret.GetOwnerReferences()
	if len(ownerReferences) != 1 {
		return nil, nil, ErrUnexpectedSecretsOwner
	}

	ownerRef := ownerReferences[0]
	if !(ownerRef.APIVersion == r.ownerReference.APIVersion &&
		ownerRef.Kind == r.ownerReference.Kind &&
		ownerRef.Name == r.ownerReference.Name &&
		ownerRef.UID == r.ownerReference.UID) {
		return nil, nil, ErrUnexpectedSecretsOwner
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

// putServiceCertificates is idempotent but not atomic in sense that on error some secrets might be persisted
// while others are not.
// Edge cases:
// - Fails for certificates with a service type that doesn't appear in r.secrets, as we don't know where to store them.
// - Not all services types in r.secrets are required to appear in certificates, missing service types are just skipped.
func (r *serviceCertificatesRepoSecretsImpl) putServiceCertificates(ctx context.Context,
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

	patch := []patchSecretDataByteMap{{
		Op:    "replace",
		Path:  "/data",
		Value: r.secretDataForCertificate(secretSpec, caPem, cert),
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

// createSecrets creates the k8s secrets for initialCertificates.
// Each secret is created with the owner specified in the constructor as owner, and with initialCertificates stored
// in the secret data.
// This only creates secrets for the service types that appear in initialCertificates.
func (r *serviceCertificatesRepoSecretsImpl) createSecrets(ctx context.Context,
	initialCertificates *storage.TypedServiceCertificateSet) error {

	var createErr error
	for _, certificate := range initialCertificates.GetServiceCerts() {
		secretSpec, ok := r.secrets[certificate.GetServiceType()]
		if !ok {
			err := errors.Errorf("unkown service type %q", certificate.GetServiceType())
			createErr = multierror.Append(createErr, err)
		} else {
			_, createSecretErr := r.createSecret(ctx, initialCertificates.GetCaPem(), certificate, secretSpec)
			if createSecretErr != nil {
				err := errors.Wrapf(createSecretErr, errForServiceFormat, certificate.GetServiceType())
				createErr = multierror.Append(createErr, err)
			}
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return createErr
}

func (r *serviceCertificatesRepoSecretsImpl) createSecret(ctx context.Context, caPem []byte,
	certificate *storage.TypedServiceCertificate, secretSpec serviceCertSecretSpec) (*v1.Secret, error) {
	return r.secretsClient.Create(ctx, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            secretSpec.secretName,
			Namespace:       r.namespace,
			OwnerReferences: []metav1.OwnerReference{r.ownerReference},
		},
		Data: r.secretDataForCertificate(secretSpec, caPem, certificate),
	}, metav1.CreateOptions{})
}

func (r *serviceCertificatesRepoSecretsImpl) secretDataForCertificate(secretSpec serviceCertSecretSpec, caPem []byte,
	cert *storage.TypedServiceCertificate) map[string][]byte {
	return map[string][]byte{
		secretSpec.caCertFileName:      caPem,
		secretSpec.serviceCertFileName: cert.GetCert().GetCertPem(),
		secretSpec.serviceKeyFileName:  cert.GetCert().GetKeyPem(),
	}
}
