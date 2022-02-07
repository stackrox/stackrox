package localscanner

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	certsDescription               = "local scanner credentials"
	_                certRefresher = (*certRefresherImpl)(nil)
	// ErrEmptyCertificate TODO: replace by ROX-9129
	ErrEmptyCertificate = errors.New("empty certificate")
)

type certRefresher interface {
	Start() error
	Stop()
}

// newCertRefresher returns a new certRefresher that uses `requestCertificates` to fetch certificates,
// with the timeout and backoff strategy specified, and the specified repository for persistence.
// Once started, the certRefresher will periodically refresh the certificates before expiration.
func newCertRefresher(requestCertificates requestCertificatesFunc, timeout time.Duration,
	backoff wait.Backoff, repository PutServiceCertificates) *certRefresherImpl {
	refresher := &certRefresherImpl{
		requestCertificates: requestCertificates,
		getCertsRenewalTime: GetCertsRenewalTime,
		repository:          repository,
	}
	refresher.createTicker = func() concurrency.RetryTicker {
		ticker := concurrency.NewRetryTicker(refresher.RefreshCertificates, timeout, backoff)
		return ticker
	}
	return refresher
}

type certRefresherImpl struct {
	requestCertificates requestCertificatesFunc
	getCertsRenewalTime func(certificates *storage.TypedServiceCertificateSet) (time.Time, error)
	repository          PutServiceCertificates
	createTicker        func() concurrency.RetryTicker
	ticker              concurrency.RetryTicker
}

type requestCertificatesFunc func(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error)

// GetCertsRenewalTime TODO: replace by ROX-9129
func GetCertsRenewalTime(certificates *storage.TypedServiceCertificateSet) (time.Time, error) {
	return time.UnixMilli(0), nil
}

// PutServiceCertificates TODO replace by serviceCertificatesRepo from ROX-9128
type PutServiceCertificates interface {
	// GetServiceCertificates retrieves the certificates from permanent storage.
	GetServiceCertificates(ctx context.Context) (*storage.TypedServiceCertificateSet, error)
	// PutServiceCertificates persists the certificates on permanent storage.
	PutServiceCertificates(ctx context.Context, certificates *storage.TypedServiceCertificateSet) error
}

// END TODO replace by certSecretsRepo from ROX-9128

func (r *certRefresherImpl) Start() error {
	r.Stop()
	r.ticker = r.createTicker()
	return r.ticker.Start()
}

func (r *certRefresherImpl) Stop() {
	if r.ticker != nil {
		r.ticker.Stop()
		// so ticker is stopped once
		r.ticker = nil
	}
}

// RefreshCertificates determines refreshes the certificate secrets if needed, and returns the time
// until the next refresh.
func (r *certRefresherImpl) RefreshCertificates(ctx context.Context) (timeToNextRefresh time.Duration, err error) {
	timeToNextRefresh, err = r.refreshCertificates(ctx)
	if err != nil {
		log.Errorf("refreshing %s: %s", certsDescription, err)
		return 0, err
	}

	log.Infof("successfully refreshed %v", certsDescription)
	log.Infof("%v scheduled to be refreshed in %s", certsDescription, timeToNextRefresh)
	return timeToNextRefresh, err
}

func (r *certRefresherImpl) refreshCertificates(ctx context.Context) (timeToNextRefresh time.Duration, err error) {
	certificates, fetchErr := r.repository.GetServiceCertificates(ctx)
	if fetchErr != nil {
		return 0, fetchErr
	}

	var timeToRefreshErr error
	timeToNextRefresh, timeToRefreshErr = r.getTimeToRefresh(certificates)
	if timeToRefreshErr != nil {
		return 0, timeToRefreshErr
	}
	if timeToNextRefresh > 0 {
		return timeToNextRefresh, nil
	}

	response, requestErr := r.requestCertificates(ctx)
	if requestErr != nil {
		return 0, requestErr
	}
	if response.GetError() != nil {
		return 0, errors.Errorf("central refused to issue certificates: %s", response.GetError().GetMessage())
	}
	certificates = response.GetCertificates()

	if putErr := r.repository.PutServiceCertificates(ctx, certificates); putErr != nil {
		return 0, putErr
	}

	return r.getTimeToRefresh(certificates)
}

func (r *certRefresherImpl) getTimeToRefresh(certificates *storage.TypedServiceCertificateSet) (time.Duration, error) {
	renewalTime, renewalTimeErr := r.getCertsRenewalTime(certificates)
	if renewalTimeErr == ErrEmptyCertificate {
		log.Errorf("some local scanner certificate is empty, will refresh certificates immediately")
		return 0, nil
	}
	if renewalTimeErr != nil {
		return 0, renewalTimeErr
	}

	return time.Until(renewalTime), nil
}
