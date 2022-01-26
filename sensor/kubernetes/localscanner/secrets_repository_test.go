package localscanner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	k8sTesting "k8s.io/client-go/testing"
)

const (
	namespace = "namespace"
)
var (
	forcedErr = errors.New("forced error")
	serviceTypes    = []storage.ServiceType{
		storage.ServiceType_SENSOR_SERVICE,
		storage.ServiceType_SCANNER_SERVICE,
		storage.ServiceType_SCANNER_DB_SERVICE,
		storage.ServiceType_CENTRAL_SERVICE,
	}
	capTime         = 100 * time.Millisecond
	shortBackoff         = wait.Backoff{
		Duration: capTime,
		Factor:   1,
		Jitter:   0,
		Steps:    2,
		Cap:      capTime,
	}
	longBackoff = wait.Backoff{
		Duration: 2 * time.Second,
		Factor: 10,
		Steps: 20,
	}
)
func TestCertSecretsRepo(t *testing.T) {
	suite.Run(t, new(certSecretsRepoSuite))
}

type certSecretsRepoSuite struct {
	suite.Suite
}

func (s *certSecretsRepoSuite) TestGet() {
	testCases := map[string]struct{
		expectedErr error
		f *certSecretsRepoFixture
	}{
		"successful get": {nil, s.newFixture("", shortBackoff,"foo")},
		"failed get": {forcedErr, s.newFixture("get", shortBackoff, "foo")},
		"cancelled get": {context.Canceled, s.newFixture("get", longBackoff, "foo")},
	}
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			getCtx, cancelGetCtx := context.WithCancel(context.Background())
			defer cancelGetCtx()
			doneErrSig := concurrency.NewErrorSignal()

			go func() {
				secrets, err := tc.f.repo.getSecrets(getCtx)
				if tc.expectedErr == nil {
					s.Equal(len(tc.f.secretsMap), len(secrets))
					for k, v := range tc.f.secretsMap {
						s.Equal(v, secrets[k])
					}
				}
				doneErrSig.SignalWithError(err)
			}()
			if tc.expectedErr == context.Canceled {
				go cancelGetCtx()
			}

			timeoutCtx, cancelTimeoutCtx := context.WithTimeout(context.Background(), time.Second)
			defer cancelTimeoutCtx()
			err, ok := doneErrSig.WaitUntil(timeoutCtx)
			s.Require().True(ok)
			s.checkExpectedError(tc.expectedErr, err)
		})
	}
}

func (s *certSecretsRepoSuite) TestPut() {
	testCases := map[string]struct{
		expectedErr error
		f *certSecretsRepoFixture
	}{
		"successful put": {nil, s.newFixture("", shortBackoff,"foo")},
		"failed put": {forcedErr, s.newFixture("update", shortBackoff, "foo")},
		"cancelled put": {context.Canceled, s.newFixture("update", longBackoff, "foo")},
	}
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			putCtx, cancelPutCtx := context.WithCancel(context.Background())
			defer cancelPutCtx()
			doneErrSig := concurrency.NewErrorSignal()

			go func() {
				err := tc.f.repo.putSecrets(putCtx, tc.f.secretsMap)
				doneErrSig.SignalWithError(err)
			}()
			if tc.expectedErr == context.Canceled {
				go cancelPutCtx()
			}

			timeoutCtx, cancelTimeoutCtx := context.WithTimeout(context.Background(), time.Second)
			defer cancelTimeoutCtx()
			err, ok := doneErrSig.WaitUntil(timeoutCtx)
			s.Require().True(ok)
			s.checkExpectedError(tc.expectedErr, err)
		})
	}
}

func  (s *certSecretsRepoSuite) checkExpectedError(expectedErr, err error) {
	if expectedErr != forcedErr {
		s.Equal(expectedErr, err)
	} else {
		// multierror wraps forcedErr
		s.NotNil(err)
	}
}

type certSecretsRepoFixture struct {
	repo certSecretsRepo
	secretClient corev1.SecretInterface
	secretsMap map[storage.ServiceType]*v1.Secret
}

func (s *certSecretsRepoSuite) newFixture(verbToError string, backoff wait.Backoff, secretNames ...string) *certSecretsRepoFixture {
	s.Require().LessOrEqual(len(secretNames), len(serviceTypes))

	secretsNamesMap := make(map[storage.ServiceType]string, len(secretNames))
	secretsMap := make(map[storage.ServiceType]*v1.Secret, len(secretNames))
	objects := make([]runtime.Object, len(secretNames))
	for i, secretName := range secretNames {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretName,
				Namespace: namespace,
			},
		}
		secretsNamesMap[serviceTypes[i]] = secretName
		secretsMap[serviceTypes[i]] = secret
		objects[i] = secret
	}
	clientSet := fake.NewSimpleClientset(objects...)
	secretsClient := clientSet.CoreV1().Secrets(namespace)
	clientSet.CoreV1().(*fakecorev1.FakeCoreV1).PrependReactor(verbToError, "secrets", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &v1.Secret{}, forcedErr
	})
	return &certSecretsRepoFixture{
		repo: NewCertSecretsRepo(secretsNamesMap, backoff, secretsClient),
		secretClient: secretsClient,
		secretsMap: secretsMap,
	}
}
