package certrepo

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/sensor/utils"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	log = logging.LoggerForModule()

	// ErrUnexpectedSecretsOwner indicates that this repository should not be updating the certificates in
	// the secrets they do not have the expected owner.
	ErrUnexpectedSecretsOwner = errors.New("unexpected owner for certificate secrets")

	// ErrDifferentCAForDifferentServiceTypes indicates that different service types have different values
	// for CA stored in their secrets.
	ErrDifferentCAForDifferentServiceTypes = errors.New("found different CA PEM in secret Data for different service types")

	// ErrMissingSecretData indicates some secret has no data.
	ErrMissingSecretData = errors.New("missing secret data")

	errForServiceFormat = "for service type %q"

	_ ServiceCertificatesRepo = (*ServiceCertificatesRepoSecrets)(nil)
)

// ServiceCertificatesRepoSecrets is a ServiceCertificatesRepo that uses k8s secrets for persistence.
type ServiceCertificatesRepoSecrets struct {
	Secrets        map[storage.ServiceType]ServiceCertSecretSpec
	OwnerReference metav1.OwnerReference
	Namespace      string
	SecretsClient  corev1.SecretInterface
}

// ServiceCertSecretSpec specifies the name of the secret where certificates for a service are stored, and
// the secret data keys where each certificate file is stored.
type ServiceCertSecretSpec struct {
	SecretName          string
	CaCertFileName      string
	ServiceCertFileName string
	ServiceKeyFileName  string
}

// NewServiceCertSecretSpec creates a ServiceCertSecretSpec with default filenames
func NewServiceCertSecretSpec(secretName string) ServiceCertSecretSpec {
	return ServiceCertSecretSpec{
		SecretName:          secretName,
		CaCertFileName:      mtls.CACertFileName,
		ServiceCertFileName: mtls.ServiceCertFileName,
		ServiceKeyFileName:  mtls.ServiceKeyFileName,
	}
}

// GetServiceCertificates fails as soon as the context is cancelled. Otherwise it returns a multierror that can contain
// the following errors:
// - ErrUnexpectedSecretsOwner in case the owner specified in the constructor is not the sole owner of all secrets.
// - ErrMissingSecretData in case any secret has no data.
// - ErrDifferentCAForDifferentServiceTypes in case the CA is not the same in all secrets.
// If the data for a secret is missing some expecting key then the corresponding field in the TypedServiceCertificate.
// for that secret will contain a zero value.
func (r *ServiceCertificatesRepoSecrets) GetServiceCertificates(ctx context.Context) (*storage.TypedServiceCertificateSet, error) {
	certificates := &storage.TypedServiceCertificateSet{}
	certificates.ServiceCerts = make([]*storage.TypedServiceCertificate, 0)
	var getErr error
	for serviceType, secretSpec := range r.Secrets {
		// on context cancellation abort getting other secrets.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		certificate, ca, err := r.getServiceCertificate(ctx, serviceType, secretSpec)
		if err != nil {
			getErr = multierror.Append(getErr, err)
			continue
		}
		if certificates.GetCaPem() == nil {
			certificates.CaPem = ca
		} else {
			if !bytes.Equal(certificates.GetCaPem(), ca) {
				getErr = multierror.Append(getErr, ErrDifferentCAForDifferentServiceTypes)
				continue
			}
		}
		certificates.ServiceCerts = append(certificates.ServiceCerts, certificate)
	}

	if getErr != nil {
		return nil, getErr
	}

	return certificates, nil
}

func (r *ServiceCertificatesRepoSecrets) getServiceCertificate(ctx context.Context, serviceType storage.ServiceType,
	secretSpec ServiceCertSecretSpec) (cert *storage.TypedServiceCertificate, ca []byte, err error) {

	secret, getErr := r.SecretsClient.Get(ctx, secretSpec.SecretName, metav1.GetOptions{})
	if getErr != nil {
		return nil, nil, getErr
	}

	ownerReferences := secret.GetOwnerReferences()
	if len(ownerReferences) != 1 {
		return nil, nil, ErrUnexpectedSecretsOwner
	}

	if ownerReferences[0].UID != r.OwnerReference.UID {
		return nil, nil, ErrUnexpectedSecretsOwner
	}

	secretData := secret.Data
	if secretData == nil {
		return nil, nil, ErrMissingSecretData
	}

	return &storage.TypedServiceCertificate{
		ServiceType: serviceType,
		Cert: &storage.ServiceCertificate{
			CertPem: secretData[secretSpec.ServiceCertFileName],
			KeyPem:  secretData[secretSpec.ServiceKeyFileName],
		},
	}, secretData[secretSpec.CaCertFileName], nil
}

// EnsureServiceCertificates ensures the k8s secrets for the services exist, and that they contain the certificates
// in their data.
// This operation is idempotent but not atomic in sense that on error some secrets might be created and updated,
// while others are not.
// The first return value contains the certificates that have been persisted.
// Each missing secret is created with the owner specified in the constructor as owner.
// This only creates secrets for the service types that appear in certificates, missing service types are just skipped.
// Fails for certificates with a service type that doesn't appear in r.Secrets, as we don't know where to store them.
func (r *ServiceCertificatesRepoSecrets) EnsureServiceCertificates(ctx context.Context,
	certificates *storage.TypedServiceCertificateSet) ([]*storage.TypedServiceCertificate, error) {
	caPem := certificates.GetCaPem()
	persistedCertificates := make([]*storage.TypedServiceCertificate, 0, len(certificates.GetServiceCerts()))
	var serviceErrors error
	for _, cert := range certificates.GetServiceCerts() {
		// on context cancellation abort putting other secrets.
		if ctx.Err() != nil {
			return persistedCertificates, ctx.Err()
		}

		secretSpec, ok := r.Secrets[cert.GetServiceType()]
		if !ok {
			log.Warnf("skipping persisting of certificate for unknown service type: %q", cert.GetServiceType())
			continue
		}
		if err := r.ensureServiceCertificate(ctx, caPem, cert, secretSpec); err != nil {
			serviceErrors = multierror.Append(serviceErrors, err)
		} else {
			persistedCertificates = append(persistedCertificates, cert)
		}
	}

	return persistedCertificates, serviceErrors
}

func (r *ServiceCertificatesRepoSecrets) ensureServiceCertificate(ctx context.Context, caPem []byte,
	cert *storage.TypedServiceCertificate, secretSpec ServiceCertSecretSpec) error {
	patchErr := r.patchServiceCertificate(ctx, caPem, cert, secretSpec)
	if k8sErrors.IsNotFound(patchErr) {
		_, createErr := r.createSecret(ctx, caPem, cert, secretSpec)
		return createErr
	}
	return patchErr
}

func (r *ServiceCertificatesRepoSecrets) patchServiceCertificate(ctx context.Context, caPem []byte,
	cert *storage.TypedServiceCertificate, secretSpec ServiceCertSecretSpec) error {
	patch := []patchSecretDataByteMap{{
		Op:    "replace",
		Path:  "/data",
		Value: r.secretDataForCertificate(secretSpec, caPem, cert),
	}, {Op: "replace",
		Path:  "/metadata/labels",
		Value: utils.GetTLSSecretLabels(),
	}}
	patchBytes, marshallingErr := json.Marshal(patch)
	if marshallingErr != nil {
		return errors.Wrapf(marshallingErr, errForServiceFormat, cert.GetServiceType())
	}
	if _, patchErr := r.SecretsClient.Patch(ctx, secretSpec.SecretName, k8sTypes.JSONPatchType, patchBytes,
		metav1.PatchOptions{}); patchErr != nil {
		return errors.Wrapf(patchErr, errForServiceFormat, cert.GetServiceType())
	}

	return nil
}

type patchSecretDataByteMap struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

func (r *ServiceCertificatesRepoSecrets) createSecret(ctx context.Context, caPem []byte,
	certificate *storage.TypedServiceCertificate, secretSpec ServiceCertSecretSpec) (*v1.Secret, error) {

	return r.SecretsClient.Create(ctx, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            secretSpec.SecretName,
			Namespace:       r.Namespace,
			Labels:          utils.GetTLSSecretLabels(),
			Annotations:     utils.GetSensorKubernetesAnnotations(),
			OwnerReferences: []metav1.OwnerReference{r.OwnerReference},
		},
		Data: r.secretDataForCertificate(secretSpec, caPem, certificate),
	}, metav1.CreateOptions{})
}

func (r *ServiceCertificatesRepoSecrets) secretDataForCertificate(secretSpec ServiceCertSecretSpec, caPem []byte,
	cert *storage.TypedServiceCertificate) map[string][]byte {

	return map[string][]byte{
		secretSpec.CaCertFileName:      caPem,
		secretSpec.ServiceCertFileName: cert.GetCert().GetCertPem(),
		secretSpec.ServiceKeyFileName:  cert.GetCert().GetKeyPem(),
	}
}
