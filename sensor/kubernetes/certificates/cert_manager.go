package certificates

import (
	"context"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/retry"
)

const (
	// FIXME adjust
	internalChannelBuffSize   = 50
	defaultCertRequestTimeout = time.Minute
)

var (
	log = logging.LoggerForModule()
	// FIXME adjust
	k8sAPIBackoff                           = retry.DefaultBackoff
	_             secretsExpirationStrategy = (*secretsExpirationStrategyImpl)(nil)

	_ CertManager = (*certManagerImpl)(nil)
)

type secretsExpirationStrategy interface {
	GetSecretsDuration(secrets map[storage.ServiceType]*v1.Secret) time.Duration
}

type errorReporter interface {
	Report(err error)
}

type jobScheduler interface {
	AfterFunc(d time.Duration, f func()) *time.Timer
}

// CertManager is in charge of storing and refreshing service TLS certificates in a set of k8s secrets.
type CertManager interface {
	Start(ctx context.Context) error
	Stop()
	// HandleIssueCertificatesResponse handles a certificate issue response.
	// - Precondition: if issueError is nil then certificates is not nil.
	// - Implementations should handle a nil receiver like an unknown request ID.
	HandleIssueCertificatesResponse(requestID string, issueError error, certificates *storage.TypedServiceCertificateSet) error
}

// CertIssuanceFunc can be used to request a certificate.
type CertIssuanceFunc func(CertManager) (requestID string, err error)
type certManagerImpl struct {
	// should be kept constant.
	secretNames             map[storage.ServiceType]string
	secretsClient           corev1.SecretInterface
	issueCerts              CertIssuanceFunc
	stopC                   concurrency.ErrorSignal
	certRequestTimeout      time.Duration
	certRequestBackoffProto wait.Backoff
	expirationStrategy      secretsExpirationStrategy
	errorReporter           errorReporter
	jobScheduler            jobScheduler
	// set at Start().
	ctx context.Context
	// handled by loop goroutine.
	dispatchC                    chan interface{}
	requestStatus                *requestStatus
	refreshTimer                 *time.Timer
	certIssueRequestTimeoutTimer *time.Timer
}

type requestStatus struct {
	requestID string
	backoff   wait.Backoff
}

// NewCertManager creates a new CertManager.
func NewCertManager(secretsClient corev1.SecretInterface, secretNames map[storage.ServiceType]string,
	certRequestBackoff wait.Backoff, issueCerts CertIssuanceFunc) CertManager {
	return newCertManager(secretsClient, secretNames, certRequestBackoff, issueCerts)
}

func newCertManager(secretsClient corev1.SecretInterface, secretNames map[storage.ServiceType]string,
	certRequestBackoff wait.Backoff, issueCerts CertIssuanceFunc) *certManagerImpl {
	return &certManagerImpl{
		secretNames:             secretNames,
		secretsClient:           secretsClient,
		issueCerts:              issueCerts,
		stopC:                   concurrency.NewErrorSignal(),
		certRequestTimeout:      defaultCertRequestTimeout,
		certRequestBackoffProto: certRequestBackoff,
		expirationStrategy:      &secretsExpirationStrategyImpl{},
		errorReporter:           &errorReporterImpl{},
		jobScheduler:            &jobSchedulerImpl{},
		dispatchC:               make(chan interface{}, internalChannelBuffSize),
		requestStatus:           &requestStatus{},
	}
}

func (c *certManagerImpl) Start(ctx context.Context) error {
	c.ctx = ctx
	secrets, err := c.fetchSecrets()
	if err != nil {
		return errors.Wrapf(err, "fetching secrets %v", c.secretNames)
	}
	// this refreshes immediately if certificates are already expired.
	c.scheduleIssueCertificatesRefresh(c.expirationStrategy.GetSecretsDuration(secrets))

	go c.loop()

	return nil
}

func (c *certManagerImpl) Stop() {
	c.stopC.Signal()
}

func (c *certManagerImpl) loop() {
	// FIXME: protect private methods and fields
	for {
		select {
		case msg := <-c.dispatchC:
			switch m := msg.(type) {
			case requestCertificates:
				c.errorReporter.Report(c.requestCertificates())
			case handleIssueCertificatesResponse:
				c.errorReporter.Report(c.handleIssueCertificatesResponse(m.requestID, m.issueError, m.certificates))
			case issueCertificatesTimeout:
				c.errorReporter.Report(c.issueCertificatesTimeout(m.requestID))
			default:
				c.errorReporter.Report(errors.Errorf("received unknown message %v, message will be ignored", msg))
			}

		case <-c.stopC.Done():
			c.errorReporter.Report(c.doStop())
			return
		}
	}
}

type handleIssueCertificatesResponse struct {
	requestID    string
	issueError   error
	certificates *storage.TypedServiceCertificateSet
}

type requestCertificates struct{}

type issueCertificatesTimeout struct {
	requestID string
}

func (c *certManagerImpl) setRefreshTimer(timer *time.Timer) {
	if c.refreshTimer != nil {
		c.refreshTimer.Stop()
	}
	c.refreshTimer = timer
}

func (c *certManagerImpl) setCertIssueRequestTimeoutTimer(timer *time.Timer) {
	if c.certIssueRequestTimeoutTimer != nil {
		c.certIssueRequestTimeoutTimer.Stop()
	}
	c.certIssueRequestTimeoutTimer = timer
}

// set request id, and stops timers.
func (c *certManagerImpl) setRequestID(requestID string) {
	c.requestStatus.requestID = requestID
	c.setRefreshTimer(nil)
	c.setCertIssueRequestTimeoutTimer(nil)
}

// reset retry backoff.
func (c *certManagerImpl) resetBackoff() {
	c.requestStatus.backoff = c.certRequestBackoffProto
}

func (c *certManagerImpl) HandleIssueCertificatesResponse(requestID string, issueError error, certificates *storage.TypedServiceCertificateSet) error {
	if c == nil {
		return errors.Errorf("unknown request ID %s, potentially due to request timeout", requestID)
	}
	c.dispatchC <- handleIssueCertificatesResponse{requestID: requestID, issueError: issueError, certificates: certificates}
	return nil
}

// should only be called from the loop goroutine.
func (c *certManagerImpl) requestCertificates() error {
	requestID, err := c.issueCerts(c)
	if err != nil {
		// client side error
		c.scheduleRetryIssueCertificatesRefresh()
		return errors.Wrapf(err, "client error sending request to issue certificates for secrets %v",
			c.secretNames)
	}
	c.setRequestID(requestID)
	c.setCertIssueRequestTimeoutTimer(c.jobScheduler.AfterFunc(c.certRequestTimeout, func() {
		log.Debugf("request with id %q will timeout in %s", requestID, c.certRequestTimeout)
		c.dispatchC <- issueCertificatesTimeout{requestID: requestID}
	}))
	return nil
}

// should only be called from the loop goroutine.
func (c *certManagerImpl) handleIssueCertificatesResponse(requestID string, issueError error,
	certificates *storage.TypedServiceCertificateSet) error {
	if requestID != c.requestStatus.requestID {
		// silently ignore responses sent to the wrong CertManager.
		log.Debugf("ignoring issue certificate response from unknown request id %q", requestID)
		return nil
	}

	if issueError != nil {
		// server side error.
		c.scheduleRetryIssueCertificatesRefresh()
		return errors.Wrapf(issueError, "server side error issuing certificates for secrets %v", c.secretNames)
	}

	nextTimeToRefresh, refreshErr := c.refreshSecrets(certificates)
	if refreshErr != nil {
		c.scheduleRetryIssueCertificatesRefresh()
		return errors.Wrapf(refreshErr, "failure to store the new certificates in the secrets %v", c.secretNames)
	}

	log.Infof("successfully refreshed credential in secrets %v", c.secretNames)
	c.resetBackoff()
	c.scheduleIssueCertificatesRefresh(nextTimeToRefresh)
	return nil
}

// should only be called from the loop goroutine.
func (c *certManagerImpl) issueCertificatesTimeout(requestID string) error {
	if requestID != c.requestStatus.requestID {
		// this is a timeout for a request we don't care about anymore.
		log.Debugf("ignoring timeout on issue certificate request from unknown request id %q", requestID)
		return nil
	}
	c.scheduleRetryIssueCertificatesRefresh()
	return errors.Errorf("timeout waiting for certificates for secrets %v on request with id %q after waiting "+
		"for %s", c.secretNames, requestID, c.certRequestTimeout)
}

// should only be called from the loop goroutine.
func (c *certManagerImpl) doStop() error {
	c.setRequestID("")
	log.Infof("cert manager for secrets %v stopped.", c.secretNames) // FIXME
	return nil
}

func (c *certManagerImpl) scheduleRetryIssueCertificatesRefresh() {
	c.scheduleIssueCertificatesRefresh(c.requestStatus.backoff.Step())
}

func (c *certManagerImpl) scheduleIssueCertificatesRefresh(timeToRefresh time.Duration) {
	log.Infof("certificates for secrets %v scheduled to be refreshed in %s", c.secretNames, timeToRefresh)
	// ignore eventual responses for this request.
	c.setRequestID("")
	c.setRefreshTimer(c.jobScheduler.AfterFunc(timeToRefresh, func() {
		c.dispatchC <- requestCertificates{}
	}))
}

func (c *certManagerImpl) fetchSecrets() (map[storage.ServiceType]*v1.Secret, error) {
	secretsMap := make(map[storage.ServiceType]*v1.Secret, len(c.secretNames))
	var fetchErr error
	for serviceType, secretName := range c.secretNames {
		var (
			secret *v1.Secret
			err    error
		)
		retryErr := retry.OnError(k8sAPIBackoff,
			func(err error) bool {
				return !k8sErrors.IsNotFound(err)
			},
			func() error {
				secret, err = c.secretsClient.Get(c.ctx, secretName, metav1.GetOptions{})
				return err
			},
		)
		if retryErr != nil {
			fetchErr = multierror.Append(fetchErr, errors.Wrapf(retryErr, "for secret %s", secretName))
		} else {
			secretsMap[serviceType] = secret
		}
	}

	if fetchErr != nil {
		return nil, fetchErr
	}
	return secretsMap, nil
}

// Performs retries for reads and writes with the k8s API.
// On success, it returns the duration after which the secrets should be refreshed.
func (c *certManagerImpl) refreshSecrets(certificates *storage.TypedServiceCertificateSet) (time.Duration, error) {
	secrets, err := c.fetchSecrets()
	if err != nil {
		// FIXME wrap
		return 0, err
	}
	// TODO update secrets ROX-9014

	return c.expirationStrategy.GetSecretsDuration(secrets), nil
}

type secretsExpirationStrategyImpl struct{}

func (s *secretsExpirationStrategyImpl) GetSecretsDuration(secrets map[storage.ServiceType]*v1.Secret) time.Duration {
	// TODO ROX-9014
	return 5 * time.Second
}

type errorReporterImpl struct{}

func (*errorReporterImpl) Report(err error) {
	if err != nil {
		log.Error(err)
	}
}

type jobSchedulerImpl struct{}

func (*jobSchedulerImpl) AfterFunc(d time.Duration, f func()) *time.Timer {
	return time.AfterFunc(d, f)
}
