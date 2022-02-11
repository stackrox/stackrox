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
		"successful get": {expectedErr: nil, fixture: s.newFixture("")},
		"failed get":     {expectedErr: errForced, fixture: s.newFixture("get")},
		"cancelled get":  {expectedErr: context.Canceled, fixture: s.newFixture("")},
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
		expectError  bool
		secondCASize int
	}{
		"same CAs successful get":  {expectError: false, secondCASize: 0},
		"different CAs failed get": {expectError: true, secondCASize: 1},
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
			if tc.expectError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *serviceCertificatesRepoSecretsImplSuite) TestPut() {
	testCases := map[string]struct {
		expectedErr error
		fixture     *certSecretsRepoFixture
	}{
		"successful put": {expectedErr: nil, fixture: s.newFixture("")},
		"failed put":     {expectedErr: errForced, fixture: s.newFixture("patch")},
		"cancelled put":  {expectedErr: context.Canceled, fixture: s.newFixture("")},
	}
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			putCtx, cancelPutCtx := context.WithCancel(context.Background())
			defer cancelPutCtx()
			if tc.expectedErr == context.Canceled {
				cancelPutCtx()
			}

			err := tc.fixture.repo.putServiceCertificates(putCtx, tc.fixture.certificates)

			s.ErrorIs(err, tc.expectedErr)
		})
	}
}

func (s *serviceCertificatesRepoSecretsImplSuite) TestGetNoSecretDataFailure() {
	fixture := s.newFixtureAdvancedOpts("", true)

	_, err := fixture.repo.getServiceCertificates(context.Background())

	s.ErrorIs(err, ErrMissingSecretData)
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
			fixture := s.newFixtureAdvancedOpts("", false, tc.missingSecretDataKey)
			certificates, err := fixture.repo.getServiceCertificates(context.Background())
			tc.setExpectedCertsFunc(fixture.certificates)

			s.NoError(err)
			s.Equal(fixture.certificates, certificates)
		})
	}
}

func (s *serviceCertificatesRepoSecretsImplSuite) TestPutUnknownServiceTypeFailure() {
	fixture := s.newFixture("")
	s.getFirstServiceCertificate(fixture.certificates).ServiceType = anotherServiceType
	err := fixture.repo.putServiceCertificates(context.Background(), fixture.certificates)
	s.Error(err)
}

func (s *serviceCertificatesRepoSecretsImplSuite) TestPutMissingServiceTypeSuccess() {
	fixture := s.newFixture("")
	fixture.certificates.ServiceCerts = make([]*storage.TypedServiceCertificate, 0)
	err := fixture.repo.putServiceCertificates(context.Background(), fixture.certificates)
	s.NoError(err)
}

func (s *serviceCertificatesRepoSecretsImplSuite) TestCreateSecretsNoCertificatesSuccess() {
	clientSet := fake.NewSimpleClientset(sensorDeployment)
	secretsClient := clientSet.CoreV1().Secrets(namespace)
	repo := newServiceCertificatesRepo(sensorOwnerReference()[0], namespace, secretsClient)

	s.NoError(repo.createSecrets(context.Background(), nil))
}

func (s *serviceCertificatesRepoSecretsImplSuite) TestCreateSecretsCancelFailure() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	clientSet := fake.NewSimpleClientset(sensorDeployment)
	secretsClient := clientSet.CoreV1().Secrets(namespace)
	repo := newServiceCertificatesRepo(sensorOwnerReference()[0], namespace, secretsClient)

	s.Error(repo.createSecrets(ctx, certificates.Clone()))
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

func (s *serviceCertificatesRepoSecretsImplSuite) newFixture(verbToError string) *certSecretsRepoFixture {
	return s.newFixtureAdvancedOpts(verbToError, false)
}

func (s *serviceCertificatesRepoSecretsImplSuite) newFixtureAdvancedOpts(verbToError string, emptySecretData bool,
	missingSecretDataKeys ...string) *certSecretsRepoFixture {
	certificates := certificates.Clone()
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("%s-secret", serviceType),
			Namespace:       namespace,
			OwnerReferences: sensorOwnerReference(),
		},
	}
	if !emptySecretData {
		secret.Data = map[string][]byte{
			mtls.CACertFileName:      certificates.GetCaPem(),
			mtls.ServiceCertFileName: serviceCertificate.GetCert().GetCertPem(),
			mtls.ServiceKeyFileName:  serviceCertificate.GetCert().GetKeyPem(),
		}
	}
	for _, secretDataKey := range missingSecretDataKeys {
		delete(secret.Data, secretDataKey)
	}
	secrets := map[storage.ServiceType]*v1.Secret{serviceType: secret}
	clientSet := fake.NewSimpleClientset(sensorDeployment, secret)
	secretsClient := clientSet.CoreV1().Secrets(namespace)
	clientSet.CoreV1().(*fakecorev1.FakeCoreV1).PrependReactor(verbToError, "secrets", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &v1.Secret{}, errForced
	})
	repo := newTestRepo(secrets, secretsClient)
	return &certSecretsRepoFixture{
		repo:          repo,
		secretsClient: secretsClient,
		certificates:  certificates,
	}
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

	secretsSpec := make(map[storage.ServiceType]serviceCertSecretSpec)
	for serviceType, secret := range secrets {
		secretsSpec[serviceType] = serviceCertSecretSpec{
			secretName:          secret.Name,
			caCertFileName:      mtls.CACertFileName,
			serviceCertFileName: mtls.ServiceCertFileName,
			serviceKeyFileName:  mtls.ServiceKeyFileName,
		}
	}

	return &serviceCertificatesRepoSecretsImpl{
		secrets:        secretsSpec,
		ownerReference: sensorOwnerReference()[0],
		namespace:      namespace,
		secretsClient:  secretsClient,
	}
}
