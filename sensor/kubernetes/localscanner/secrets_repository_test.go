package localscanner

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	k8sTesting "k8s.io/client-go/testing"
)

const (
	namespace   = "stackrox-ns"
)

var (
	errForced = errors.New("forced error")
)

func TestCertSecretsRepo(t *testing.T) {
	suite.Run(t, new(certSecretsRepoSuite))
}

type certSecretsRepoSuite struct {
	suite.Suite
}

func (s *certSecretsRepoSuite) TestGet() {
	testCases := map[string]struct {
		expectedErr error
		f           *certSecretsRepoFixture
	}{
		"successful get": {nil, s.newFixture("", "foo")},
		"failed get":     {errForced, s.newFixture("get", "foo")},
		"cancelled get":  {context.Canceled, s.newFixture("get", "foo")},
	}
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			getCtx, cancelGetCtx := context.WithCancel(context.Background())
			defer cancelGetCtx()
			if tc.expectedErr == context.Canceled {
				cancelGetCtx()
			}

			secrets, err := tc.f.repo.getSecrets(getCtx)

			if tc.expectedErr == nil {
				s.Equal(len(tc.f.secretsMap), len(secrets))
				for k, v := range tc.f.secretsMap {
					s.Equal(v, secrets[k])
				}
			}
			s.checkExpectedError(tc.expectedErr, err)
		})
	}
}

func (s *certSecretsRepoSuite) TestPut() {
	testCases := map[string]struct {
		expectedErr error
		f           *certSecretsRepoFixture
	}{
		"successful put": {nil, s.newFixture("", "foo")},
		"failed put":     {errForced, s.newFixture("update", "foo")},
		"cancelled put":  {context.Canceled, s.newFixture("update", "foo")},
	}
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			putCtx, cancelPutCtx := context.WithCancel(context.Background())
			defer cancelPutCtx()
			if tc.expectedErr == context.Canceled {
				cancelPutCtx()
			}

			err := tc.f.repo.putSecrets(putCtx, tc.f.secretsMap)

			s.checkExpectedError(tc.expectedErr, err)
		})
	}
}

func (s *certSecretsRepoSuite) checkExpectedError(expectedErr, err error) {
	if expectedErr != errForced {
		s.Equal(expectedErr, err)
	} else {
		// multierror wraps errForced
		s.Error(err)
	}
}

type certSecretsRepoFixture struct {
	repo         certSecretsRepo
	secretClient corev1.SecretInterface
	secretsMap   map[storage.ServiceType]*v1.Secret
}

func (s *certSecretsRepoSuite) newFixture(verbToError string, secretName string) *certSecretsRepoFixture {
	serviceType := storage.ServiceType_SCANNER_SERVICE
	secretsNamesMap := map[storage.ServiceType]string{serviceType: secretName}
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
	}
	secretsMap := map[storage.ServiceType]*v1.Secret{serviceType: secret}
	clientSet := fake.NewSimpleClientset(secret)
	secretsClient := clientSet.CoreV1().Secrets(namespace)
	clientSet.CoreV1().(*fakecorev1.FakeCoreV1).PrependReactor(verbToError, "secrets", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &v1.Secret{}, errForced
	})
	return &certSecretsRepoFixture{
		repo:         newCertSecretsRepo(secretsNamesMap, secretsClient),
		secretClient: secretsClient,
		secretsMap:   secretsMap,
	}
}
