package localscanner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	k8sTesting "k8s.io/client-go/testing"
)

const (
	namespace = "namespace"
)

var (
	errForced    = errors.New("forced error")
	serviceTypes = []storage.ServiceType{
		storage.ServiceType_SENSOR_SERVICE,
		storage.ServiceType_SCANNER_SERVICE,
		storage.ServiceType_SCANNER_DB_SERVICE,
		storage.ServiceType_CENTRAL_SERVICE,
	}
	capTime      = 10 * time.Millisecond
	shortBackoff = wait.Backoff{
		Duration: capTime,
		Factor:   1,
		Jitter:   0,
		Steps:    2,
		Cap:      capTime,
	}
	longBackoff = wait.Backoff{
		Duration: capTime,
		Factor:   1,
		Steps:    10,
		Jitter:   0,
		Cap:      capTime,
	}
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
		"successful get": {nil, s.newFixture("", shortBackoff, "foo")},
		"failed get":     {errForced, s.newFixture("get", shortBackoff, "foo")},
		"cancelled get":  {context.Canceled, s.newFixture("get", longBackoff, "foo")},
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
				cancelGetCtx()
			}

			err, ok := doneErrSig.WaitWithTimeout(100 * time.Millisecond)
			s.Require().True(ok)
			s.checkExpectedError(tc.expectedErr, err)
		})
	}
}

func (s *certSecretsRepoSuite) TestPut() {
	testCases := map[string]struct {
		expectedErr error
		f           *certSecretsRepoFixture
	}{
		"successful put": {nil, s.newFixture("", shortBackoff, "foo")},
		"failed put":     {errForced, s.newFixture("update", shortBackoff, "foo")},
		"cancelled put":  {context.Canceled, s.newFixture("update", longBackoff, "foo")},
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
				cancelPutCtx()
			}

			err, ok := doneErrSig.WaitWithTimeout(100 * time.Millisecond)
			s.Require().True(ok)
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

func (s *certSecretsRepoSuite) newFixture(verbToError string, backoff wait.Backoff, secretNames ...string) *certSecretsRepoFixture {
	s.Require().LessOrEqual(len(secretNames), len(serviceTypes))

	secretsNamesMap := make(map[storage.ServiceType]string, len(secretNames))
	secretsMap := make(map[storage.ServiceType]*v1.Secret, len(secretNames))
	objects := make([]runtime.Object, len(secretNames))
	for i, secretName := range secretNames {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
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
		return true, &v1.Secret{}, errForced
	})
	return &certSecretsRepoFixture{
		repo:         NewCertSecretsRepo(secretsNamesMap, backoff, secretsClient),
		secretClient: secretsClient,
		secretsMap:   secretsMap,
	}
}
