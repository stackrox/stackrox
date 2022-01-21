package certificates

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// FIXME adjust
	defaultCertRequestTimeout = time.Minute
	refreshCrashWaitTime      = time.Minute
)

var (
	log               = logging.LoggerForModule()
	_   CertRefresher = (*certRefresherImpl)(nil)
)

// CertRefresher is in charge of scheduling the refresh of the TLS certificates of a set of services.
type CertRefresher interface {
	Start(ctx context.Context) error
	Stop()
}
type certRefresherImpl struct {
	conf         certRefresherConf
	ctx          context.Context
	refreshTimer *time.Timer
}

type certRefresherConf struct {
	certsDescription  string
	certificateSource CertificateSource
	issueCertificates func(context.Context) (*storage.TypedServiceCertificateSet, error)
}

// CertificateSource is able to fetch certificates of type *storage.TypedServiceCertificateSet, and
// to process the retrieved certificates.
type CertificateSource interface {
	// RetryableSource to fetch certificates of type *storage.TypedServiceCertificateSet.
	retry.RetryableSource
	// HandleCertificates stores the certificates in some permanent storage, and returns the time until the next
	// refresh.
	// If certificates are nil then this should initialize or retrieve the certificates from local storage,
	// and compute their next refresh time.
	HandleCertificates(certificates *storage.TypedServiceCertificateSet) (timeToRefresh time.Duration, err error)
}

// NewCertRefresher creates a new CertRefresher.
func NewCertRefresher(certsDescription string, certsSource CertificateSource,
	certRequestBackoff wait.Backoff) CertRefresher {
	return newCertRefresher(certsDescription, certsSource, certRequestBackoff)
}

func newCertRefresher(certsDescription string, certsSource CertificateSource,
	certRequestBackoff wait.Backoff) *certRefresherImpl {
	return &certRefresherImpl{
		conf: certRefresherConf{
			certsDescription:  certsDescription,
			certificateSource: certsSource,
			issueCertificates: createIssueCertificates(certsDescription, certsSource, certRequestBackoff),
		},
	}
}

// the returned function only fails if it is cancelled with its input context.
func createIssueCertificates(certsDescription string, certsSource retry.RetryableSource,
	backoff wait.Backoff) func(context.Context) (*storage.TypedServiceCertificateSet, error) {
	retriever := retry.NewRetryableSourceRetriever(backoff, defaultCertRequestTimeout)
	retriever.OnError = func(err error, timeToNextRetry time.Duration) {
		log.Errorf("error retrieving certificates %s, will retry in %s: %s",
			certsDescription, timeToNextRetry, err)
	}
	retriever.ValidateResult = func(maybeCerts interface{}) bool {
		_, ok := maybeCerts.(*storage.TypedServiceCertificateSet)
		return ok
	}
	return func(ctx context.Context) (*storage.TypedServiceCertificateSet, error) {
		retriever.Backoff = backoff // reset backoff for each retrieval.
		maybeCerts, err := retriever.Run(ctx, certsSource)
		if err != nil {
			return nil, err
		}
		certs, ok := maybeCerts.(*storage.TypedServiceCertificateSet)
		if !ok {
			// this shouldn't happen due to validation
			return nil, errors.Errorf("critical error: response %v has unexpected type", maybeCerts)
		}
		return certs, nil
	}
}

func (c *certRefresherImpl) Start(ctx context.Context) error {
	c.ctx = ctx
	err := c.initialRefresh()
	if err != nil {
		return err
	}
	go func() {
		<-c.ctx.Done()
		c.Stop()
	}()
	return nil
}

func (c *certRefresherImpl) Stop() {
	c.setRefreshTimer(nil)
	log.Infof("stopped for certificates %s", c.conf.certsDescription)
}

func (c *certRefresherImpl) initialRefresh() error {
	timeToRefresh, err := c.conf.certificateSource.HandleCertificates(nil)
	if err != nil {
		return errors.Wrapf(err, "critical error processing stored certificates %s, aborting",
			c.conf.certsDescription)
	}
	c.scheduleIssueCertificatesRefresh(timeToRefresh)
	return nil
}

func (c *certRefresherImpl) scheduleIssueCertificatesRefresh(timeToRefresh time.Duration) {
	c.setRefreshTimer(time.AfterFunc(timeToRefresh, func() {
		certificates, issueErr := c.conf.issueCertificates(c.ctx)
		if issueErr != nil {
			log.Errorf("critical error issuing certificates %s: %s",
				c.conf.certsDescription, issueErr)
			c.recoverFromRefreshCrash()
		}
		nextTimeToRefresh, handleErr := c.conf.certificateSource.HandleCertificates(certificates)
		if handleErr != nil {
			log.Errorf("critical error processing certificates %s: %s",
				c.conf.certsDescription, issueErr)
			c.recoverFromRefreshCrash()
		}

		log.Infof("successfully refreshed credentials for certificates %v", c.conf.certsDescription)
		c.scheduleIssueCertificatesRefresh(nextTimeToRefresh)
	}))
	log.Infof("credentials for %v scheduled to be refreshed in %s",
		c.conf.certsDescription, timeToRefresh)
}

func (c *certRefresherImpl) setRefreshTimer(timer *time.Timer) {
	if c.refreshTimer != nil {
		c.refreshTimer.Stop()
	}
	c.refreshTimer = timer
}

func (c *certRefresherImpl) recoverFromRefreshCrash() {
	// TODO: consider backoff here.
	c.setRefreshTimer(time.AfterFunc(refreshCrashWaitTime, func() {
		err := c.initialRefresh()
		if err != nil {
			log.Error(err)
			c.recoverFromRefreshCrash()
		}
	}))
	log.Errorf("refresh process for %s will restart in %s", c.conf.certsDescription, refreshCrashWaitTime)
}
