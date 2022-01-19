package certificates

import (
	"context"
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
)

const (
	namespace = "namespace"
	requestID = "requestID"
)

var (
	requestBackoff = wait.Backoff{
		Steps:    3,
		Duration: 10 * time.Millisecond,
		Factor:   10.0,
		Jitter:   0.1,
		Cap:      2 * time.Second,
	}
)

func TestHandler(t *testing.T) {
	suite.Run(t, new(certManagerSuite))
}

type certManagerSuite struct {
	suite.Suite
}

func fakeClientSet(secretNames ...string) *fake.Clientset {
	secrets := make([]runtime.Object, len(secretNames))
	for i, secretName := range secretNames {
		secrets[i] = &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: namespace}}
	}
	return fake.NewSimpleClientset(secrets...)
}

func fakeSecretsClient(secretNames ...string) corev1.SecretInterface {
	return fakeClientSet(secretNames...).CoreV1().Secrets(namespace)
}

type fixedSecretsExpirationStrategy struct {
	durations   []time.Duration
	invocations int
	signal      concurrency.ErrorSignal
}

func newFixedSecretsExpirationStrategy(durations ...time.Duration) *fixedSecretsExpirationStrategy {
	return &fixedSecretsExpirationStrategy{
		durations: durations,
		signal:    concurrency.NewErrorSignal(),
	}
}

// signals .signal when the last timeout is reached
func (s *fixedSecretsExpirationStrategy) GetSecretsDuration(secrets map[storage.ServiceType]*v1.Secret) (duration time.Duration) {
	s.invocations++
	if len(s.durations) <= 1 {
		s.signal.Signal()
		return s.durations[0]
	}

	duration, s.durations = s.durations[0], s.durations[1:]
	return duration
}

func (s *certManagerSuite) TestSuccessfulRefresh() {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	secretName := "foo"
	secretNames := map[storage.ServiceType]string{
		storage.ServiceType_SCANNER_DB_SERVICE: secretName,
	}
	secretsClient := fakeSecretsClient(secretName)
	certManager := newCertManager(secretsClient, secretNames, requestBackoff,
		func(manager CertManager) (requestID string, err error) {
			// FIXME nil certs
			s.Require().NoError(manager.HandleIssueCertificatesResponse(requestID, nil, nil))
			return requestID, nil
		})
	defer certManager.Stop()
	certManager.certRequestTimeout = 2 * time.Second
	expirationStrategy := newFixedSecretsExpirationStrategy(0, 2*time.Second)
	certManager.secretExpiration = expirationStrategy

	s.Require().NoError(certManager.Start(ctx))

	// FIXME: idea, add error handler fields to certManagerImpl, that processes errors in loop, and
	// make the processing functions return err. For prod it just logs; for test it keeps a slice of
	// errors or something we can inspect

	waitErr, ok := expirationStrategy.signal.WaitUntil(ctx)
	s.Require().True(ok)
	s.NoError(waitErr)

	// TODO assert certManager not stopped

	s.Empty(certManager.requestStatus.requestID)
}

/*
TODO failures:

- success
- server failure
- client failure
- timeout

in all check retries as expected
*/
