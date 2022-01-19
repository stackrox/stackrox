package certificates

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/mock"
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
	ctx         context.Context
	cancelCtx   context.CancelFunc
	errReporter *recordErrorReporter
	scheduler   *mockJobScheduler
	certManager *certManagerImpl
}

func (s *certManagerSuite) TearDownTest() {
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
	if s.certManager != nil {
		s.certManager.Stop()
	}

	log.Warn("FIXME")
}

func (s *certManagerSuite) initialize(testTimeout time.Duration,
	secretNamesMap map[storage.ServiceType]string,
	certRequestTimeout time.Duration, expirations []time.Duration,
	issueCerts CertIssuanceFunc) {
	ctx := context.Background()
	s.ctx, s.cancelCtx = context.WithTimeout(ctx, testTimeout)

	secretNames := make([]string, len(secretNamesMap))
	for _, secretName := range secretNamesMap {
		secretNames = append(secretNames, secretName)
	}
	secretsClient := fakeSecretsClient(secretNames...)

	s.errReporter = newRecordErrorReporter(3)
	s.scheduler = newMockJobScheduler()

	certManager := newCertManager(secretsClient, secretNamesMap, requestBackoff, issueCerts)
	certManager.certRequestTimeout = certRequestTimeout
	certManager.expirationStrategy = newFixedSecretsExpirationStrategy(expirations...)
	certManager.errorReporter = s.errReporter
	certManager.jobScheduler = s.scheduler
	s.certManager = certManager
}

func (s *certManagerSuite) TestSuccessfulInitialRefresh() {
	secretNames := map[storage.ServiceType]string{
		storage.ServiceType_SCANNER_DB_SERVICE: "foo",
	}
	certRequestTimeout := 3 * time.Second
	expirations := []time.Duration{0, 2 * time.Second}
	s.initialize(time.Second, secretNames, certRequestTimeout, expirations,
		// FIXME replace by mock method to assert on requestCertificates
		func(manager CertManager) (string, error) {
			requestID := uuid.NewV4().String()
			go func() {
				// TODO non nil certs ROX-9014
				s.Require().NoError(manager.HandleIssueCertificatesResponse(requestID, nil, nil))
			}()

			return requestID, nil
		})

	s.scheduler.On("AfterFunc", expirations[0], mock.Anything).Once()
	s.scheduler.On("AfterFunc", s.certManager.certRequestTimeout, mock.Anything).Once()
	s.scheduler.On("AfterFunc", expirations[1], mock.Anything).Once().Run(func(mock.Arguments) {
		s.certManager.Stop()
	})

	s.Require().NoError(s.certManager.Start(s.ctx))
	waitErr, ok := s.errReporter.signal.WaitUntil(s.ctx)
	s.Require().True(ok)
	s.NoError(waitErr)

	s.scheduler.AssertExpectations(s.T())
	// requestCertificates, handleIssueCertificatesResponse, stop
	s.Equal([]error{nil, nil, nil}, s.errReporter.errors)
	// TODO: assert timers nil, retry reset, request id nil
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
}

func newFixedSecretsExpirationStrategy(durations ...time.Duration) *fixedSecretsExpirationStrategy {
	return &fixedSecretsExpirationStrategy{
		durations: durations,
	}
}

// returns the last duration forever when it runs out of durations
func (s *fixedSecretsExpirationStrategy) GetSecretsDuration(map[storage.ServiceType]*v1.Secret) (duration time.Duration) {
	s.invocations++
	if len(s.durations) <= 1 {
		return s.durations[0]
	}

	duration, s.durations = s.durations[0], s.durations[1:]
	return duration
}

// the reporter will Signal() its signal as soon as numErrorsToSignal are reported.
type recordErrorReporter struct {
	reporter          errorReporter
	errors            []error
	numErrorsToSignal int
	signal            concurrency.ErrorSignal
}

func (r *recordErrorReporter) Report(err error) {
	r.errors = append(r.errors, err)
	r.reporter.Report(err)
	if len(r.errors) >= r.numErrorsToSignal {
		r.signal.Signal()
	}
}

func newRecordErrorReporter(numErrorsToSignal int) *recordErrorReporter {
	return &recordErrorReporter{
		reporter:          &errorReporterImpl{},
		signal:            concurrency.NewErrorSignal(),
		numErrorsToSignal: numErrorsToSignal,
	}
}

// AfterFunc records the call in the mock, and then returns AfterFunc() for the
// wrapped scheduler.
type mockJobScheduler struct {
	mock.Mock
	scheduler jobScheduler
}

func (s *mockJobScheduler) AfterFunc(d time.Duration, f func()) *time.Timer {
	s.Called(d, f)
	return s.scheduler.AfterFunc(d, f)
}

func newMockJobScheduler() *mockJobScheduler {
	return &mockJobScheduler{
		scheduler: &jobSchedulerImpl{},
	}
}

/*
TODO failures:

- success
- server failure
- client failure
- timeout
- unknown request ids
- nil cert manager

in all check retries as expected
*/
