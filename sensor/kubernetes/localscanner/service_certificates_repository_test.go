package localscanner

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stretchr/testify/suite"
	appsApiv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	k8sTesting "k8s.io/client-go/testing"
)

const (
	namespace = "stackrox-ns"
)

var (
	errForced          = errors.New("forced error")
	serviceType        = storage.ServiceType_SCANNER_SERVICE
	anotherServiceType = storage.ServiceType_SENSOR_SERVICE
	serviceCertificate = &storage.TypedServiceCertificate{
		ServiceType: serviceType,
		Cert: &storage.ServiceCertificate{
			CertPem: make([]byte, 0),
			KeyPem:  make([]byte, 1),
		},
	}
	certificates = &storage.TypedServiceCertificateSet{
		CaPem: make([]byte, 2),
		ServiceCerts: []*storage.TypedServiceCertificate{
			serviceCertificate,
		},
	}
	sensorDeployment = &appsApiv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sensor-deployment",
			Namespace: namespace,
		},
	}
)

func TestServiceCertificatesRepoSecretsImpl(t *testing.T) {
	suite.Run(t, new(serviceCertificatesRepoSecretsImplSuite))
}

type serviceCertificatesRepoSecretsImplSuite struct {
	suite.Suite
}

func (s *serviceCertificatesRepoSecretsImplSuite) TestGet() {
	testCases := map[string]struct {
		expectedErr error
		fixture     *certSecretsRepoFixture
	}{
		"successful get": {expectedErr: nil, fixture: s.newFixture(certSecretsRepoFixtureConfig{})},
		"failed get due to k8s API error": {
			expectedErr: errForced,
			fixture:     s.newFixture(certSecretsRepoFixtureConfig{k8sAPIVerbToError: "get"}),
		},
		"cancelled get": {expectedErr: context.Canceled, fixture: s.newFixture(certSecretsRepoFixtureConfig{})},
	}
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			getCtx, cancelGetCtx := context.WithCancel(context.Background())
			defer cancelGetCtx()
			if tc.expectedErr == context.Canceled {
				cancelGetCtx()
			}

			certificates, err := tc.fixture.repo.getServiceCertificates(getCtx)

			if tc.expectedErr == nil {
				s.Equal(tc.fixture.certificates, certificates)
			}
			s.ErrorIs(err, tc.expectedErr)
		})
	}
}

func (s *serviceCertificatesRepoSecretsImplSuite) TestGetDifferentCAsFailure() {
	testCases := map[string]struct {
		expectedErr  error
		secondCASize int
	}{
		"same CAs successful get":  {expectedErr: nil, secondCASize: 0},
		"different CAs failed get": {expectedErr: ErrDifferentCAForDifferentServiceTypes, secondCASize: 1},
	}
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			secret1 := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "secret1",
					Namespace:       namespace,
					OwnerReferences: sensorOwnerReference(),
				},
				Data: map[string][]byte{
					mtls.CACertFileName: make([]byte, 0),
				},
			}
			secret2 := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "secret2",
					Namespace:       namespace,
					OwnerReferences: sensorOwnerReference(),
				},
				Data: map[string][]byte{
					mtls.CACertFileName: make([]byte, tc.secondCASize),
				},
			}
			secrets := map[storage.ServiceType]*v1.Secret{serviceType: secret1, anotherServiceType: secret2}
			clientSet := fake.NewSimpleClientset(secret1, secret2)
			secretsClient := clientSet.CoreV1().Secrets(namespace)
			repo := newTestRepo(secrets, secretsClient)

			_, err := repo.getServiceCertificates(context.Background())

			s.ErrorIs(err, tc.expectedErr)
		})
	}
}

func (s *serviceCertificatesRepoSecretsImplSuite) TestPatch() {
	testCases := map[string]struct {
		expectedErr error
		fixture     *certSecretsRepoFixture
	}{
		"successful patch": {expectedErr: nil, fixture: s.newFixture(certSecretsRepoFixtureConfig{})},
		"failed patch due to k8s API error": {
			expectedErr: errForced,
			fixture:     s.newFixture(certSecretsRepoFixtureConfig{k8sAPIVerbToError: "patch"}),
		},
		"cancelled patch": {expectedErr: context.Canceled, fixture: s.newFixture(certSecretsRepoFixtureConfig{})},
	}
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.expectedErr == context.Canceled {
				cancel()
			}

			err := tc.fixture.repo.ensureServiceCertificates(ctx, tc.fixture.certificates)

			s.ErrorIs(err, tc.expectedErr)
		})
	}
}

func (s *serviceCertificatesRepoSecretsImplSuite) TestGetNoSecretDataFailure() {
	fixture := s.newFixture(certSecretsRepoFixtureConfig{emptySecretData: true})

	_, err := fixture.repo.getServiceCertificates(context.Background())

	s.ErrorIs(err, ErrMissingSecretData)
}

func (s *serviceCertificatesRepoSecretsImplSuite) TestGetUnexpectedSecretsOwnerFailure() {
	fixture := s.newFixture(certSecretsRepoFixtureConfig{secretOwnerRefUID: "wrong owner"})

	_, err := fixture.repo.getServiceCertificates(context.Background())

	s.ErrorIs(err, ErrUnexpectedSecretsOwner)
}

func (s *serviceCertificatesRepoSecretsImplSuite) TestGetSecretDataMissingKeysSuccess() {
	testCases := map[string]struct {
		missingSecretDataKey string
		setExpectedCertsFunc func(certificates *storage.TypedServiceCertificateSet)
	}{
		"missing CA": {
			missingSecretDataKey: mtls.CACertFileName,
			setExpectedCertsFunc: func(certificates *storage.TypedServiceCertificateSet) {
				certificates.CaPem = nil
			}},
		"missing Cert": {
			missingSecretDataKey: mtls.ServiceCertFileName,
			setExpectedCertsFunc: func(certificates *storage.TypedServiceCertificateSet) {
				s.getFirstServiceCertificate(certificates).Cert.CertPem = nil
			},
		},
		"missing Key": {
			missingSecretDataKey: mtls.ServiceKeyFileName,
			setExpectedCertsFunc: func(certificates *storage.TypedServiceCertificateSet) {
				s.getFirstServiceCertificate(certificates).Cert.KeyPem = nil
			},
		},
	}
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			fixture := s.newFixture(certSecretsRepoFixtureConfig{missingSecretDataKeys: []string{tc.missingSecretDataKey}})

			certificates, err := fixture.repo.getServiceCertificates(context.Background())

			s.Require().NoError(err)
			tc.setExpectedCertsFunc(fixture.certificates)
			s.Equal(fixture.certificates, certificates)
		})
	}
}

func (s *serviceCertificatesRepoSecretsImplSuite) TestEnsureCertsUnknownServiceTypeFailure() {
	fixture := s.newFixture(certSecretsRepoFixtureConfig{})
	s.getFirstServiceCertificate(fixture.certificates).ServiceType = anotherServiceType

	err := fixture.repo.ensureServiceCertificates(context.Background(), fixture.certificates)

	s.Error(err)
}

func (s *serviceCertificatesRepoSecretsImplSuite) TestEnsureCertsMissingServiceTypeSuccess() {
	fixture := s.newFixture(certSecretsRepoFixtureConfig{})
	fixture.certificates.ServiceCerts = make([]*storage.TypedServiceCertificate, 0)

	err := fixture.repo.ensureServiceCertificates(context.Background(), fixture.certificates)

	s.NoError(err)
}

func (s *serviceCertificatesRepoSecretsImplSuite) TestCreateSecretsNoCertificatesSuccess() {
	clientSet := fake.NewSimpleClientset(sensorDeployment)
	secretsClient := clientSet.CoreV1().Secrets(namespace)
	repo := newServiceCertificatesRepo(sensorOwnerReference()[0], namespace, secretsClient)

	s.NoError(repo.ensureServiceCertificates(context.Background(), nil))
}

func (s *serviceCertificatesRepoSecretsImplSuite) TestCreateSecretsCancelFailure() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	clientSet := fake.NewSimpleClientset(sensorDeployment)
	secretsClient := clientSet.CoreV1().Secrets(namespace)

	repo := newServiceCertificatesRepo(sensorOwnerReference()[0], namespace, secretsClient)

	s.Error(repo.ensureServiceCertificates(ctx, certificates.Clone()))
}

func (s *serviceCertificatesRepoSecretsImplSuite) TestEnsureServiceCertificateMissingSecretSuccess() {
	clientSet := fake.NewSimpleClientset(sensorDeployment)
	secretsClient := clientSet.CoreV1().Secrets(namespace)
	repo := newServiceCertificatesRepo(sensorOwnerReference()[0], namespace, secretsClient)

	err := repo.ensureServiceCertificates(context.Background(), certificates)

	s.NoError(err)
}

func (s *serviceCertificatesRepoSecretsImplSuite) getFirstServiceCertificate(
	certificates *storage.TypedServiceCertificateSet) *storage.TypedServiceCertificate {
	serviceCerts := certificates.GetServiceCerts()
	s.Require().Len(serviceCerts, 1)
	return serviceCerts[0]
}

type certSecretsRepoFixture struct {
	repo          *serviceCertificatesRepoSecretsImpl
	secretsClient corev1.SecretInterface
	certificates  *storage.TypedServiceCertificateSet
}

// newFixture creates a certSecretsRepoFixture that contains:
// 1. A secrets client corresponding to a fake k8s client set such that:
//   - It is initialized to represent a cluster with sensorDeployment and a secret that contains certificates
//     on its data, or partial data according to spec.
//   - The client set will fail all operations on the HTTP verb indicated in spec.
//   - The certificates used to initialize the data of the aforementioned secret.
//   - A repository that uses that secrets client, sensorDeployment as owner, and with a single serviceCertSecretSpec
//     for the aforementioned secret in its secrets.
func (s *serviceCertificatesRepoSecretsImplSuite) newFixture(config certSecretsRepoFixtureConfig) *certSecretsRepoFixture {
	certificates := certificates.Clone()
	ownerRef := sensorOwnerReference()
	if config.secretOwnerRefUID != "" {
		ownerRef[0].UID = types.UID(config.secretOwnerRefUID)
	}
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("%s-secret", serviceType),
			Namespace:       namespace,
			OwnerReferences: ownerRef,
		},
	}
	if !config.emptySecretData {
		secret.Data = map[string][]byte{
			mtls.CACertFileName:      certificates.GetCaPem(),
			mtls.ServiceCertFileName: serviceCertificate.GetCert().GetCertPem(),
			mtls.ServiceKeyFileName:  serviceCertificate.GetCert().GetKeyPem(),
		}
	}
	for _, secretDataKey := range config.missingSecretDataKeys {
		delete(secret.Data, secretDataKey)
	}
	secrets := map[storage.ServiceType]*v1.Secret{serviceType: secret}
	clientSet := fake.NewSimpleClientset(sensorDeployment, secret)
	secretsClient := clientSet.CoreV1().Secrets(namespace)
	clientSet.CoreV1().(*fakecorev1.FakeCoreV1).PrependReactor(config.k8sAPIVerbToError, "secrets", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &v1.Secret{}, errForced
	})
	repo := newTestRepo(secrets, secretsClient)
	return &certSecretsRepoFixture{
		repo:          repo,
		secretsClient: secretsClient,
		certificates:  certificates,
	}
}

type certSecretsRepoFixtureConfig struct {
	// HTTP verb of the k8s API should for which all operations will fail in the fake k8s client set.
	// Use the zero value so all operations work.
	k8sAPIVerbToError string
	// If true then the data of the secret used to initialize the fake k8s client set will be empty.
	emptySecretData bool
	// These keys will be removed from the data keys of the secret used to initialize the fake k8s client set.
	missingSecretDataKeys []string
	// If set to a non-zero value, then the UID of the owner of the secret used to initialize the fake k8s client
	// set will take this value.
	secretOwnerRefUID string
}

func sensorOwnerReference() []metav1.OwnerReference {
	sensorDeploymentGVK := sensorDeployment.GroupVersionKind()
	blockOwnerDeletion := false
	isController := false
	return []metav1.OwnerReference{
		{
			APIVersion:         sensorDeploymentGVK.GroupVersion().String(),
			Kind:               sensorDeploymentGVK.Kind,
			Name:               sensorDeployment.GetName(),
			UID:                sensorDeployment.GetUID(),
			BlockOwnerDeletion: &blockOwnerDeletion,
			Controller:         &isController,
		},
	}
}

func newTestRepo(secrets map[storage.ServiceType]*v1.Secret,
	secretsClient corev1.SecretInterface) *serviceCertificatesRepoSecretsImpl {

	secretsSpecs := make(map[storage.ServiceType]serviceCertSecretSpec)
	for serviceType, secret := range secrets {
		secretsSpecs[serviceType] = serviceCertSecretSpec{
			secretName:          secret.Name,
			caCertFileName:      mtls.CACertFileName,
			serviceCertFileName: mtls.ServiceCertFileName,
			serviceKeyFileName:  mtls.ServiceKeyFileName,
		}
	}

	return &serviceCertificatesRepoSecretsImpl{
		secrets:        secretsSpecs,
		ownerReference: sensorOwnerReference()[0],
		namespace:      namespace,
		secretsClient:  secretsClient,
	}
}
