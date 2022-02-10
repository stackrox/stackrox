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
	certsDescription = "local scanner credentials"
)

// newCertRefresher returns a new certRefresher that uses `requestCertificates` to fetch certificates,
// with the timeout and backoff strategy specified, and the specified repository for persistence.
// Once started, the certRefresher will periodically refresh the certificates before expiration.
func newCertRefresher(requestCertificates requestCertificatesFunc, timeout time.Duration,
	backoff wait.Backoff, repository ServiceCertificatesRepo) *certRefresherImpl {

	refresher := &certRefresherImpl{
		requestCertificates: requestCertificates,
		getCertsRenewalTime: GetCertsRenewalTime,
		repository:          repository,
	}
	refresher.createTicker = func() concurrency.RetryTicker {
		return concurrency.NewRetryTicker(refresher.refreshCertificates, timeout, backoff)
	}
	return refresher
}

type certRefresherImpl struct {
	requestCertificates requestCertificatesFunc
	getCertsRenewalTime func(certificates *storage.TypedServiceCertificateSet) (time.Time, error)
	repository          ServiceCertificatesRepo
	createTicker        func() concurrency.RetryTicker
	ticker              concurrency.RetryTicker
}

type requestCertificatesFunc func(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error)

// ServiceCertificatesRepo is in charge of persisting and retrieving a set of service certificates, thus implementing
// the [repository pattern](https://martinfowler.com/eaaCatalog/repository.html) for *storage.TypedServiceCertificateSet.
type ServiceCertificatesRepo interface {
	// GetServiceCertificates retrieves the certificates from permanent storage.
	GetServiceCertificates(ctx context.Context) (*storage.TypedServiceCertificateSet, error)
	// PutServiceCertificates persists the certificates on permanent storage.
	PutServiceCertificates(ctx context.Context, certificates *storage.TypedServiceCertificateSet) error
}

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

// refreshCertificates determines refreshes the certificate secrets if needed, and returns the time
// until the next refresh.
func (r *certRefresherImpl) refreshCertificates(ctx context.Context) (timeToNextRefresh time.Duration, err error) {
	timeToNextRefresh, err = r.ensureCertificatesAreFresh(ctx)
	if err != nil {
		log.Errorf("refreshing %s: %s", certsDescription, err)
		return 0, err
	}

	log.Infof("%v scheduled to be refreshed in %s", certsDescription, timeToNextRefresh)
	return timeToNextRefresh, err
}

func (r *certRefresherImpl) ensureCertificatesAreFresh(ctx context.Context) (timeToNextRefresh time.Duration, err error) {
	certificates, fetchErr := r.repository.GetServiceCertificates(ctx)
	if fetchErr != nil {
		return 0, fetchErr
	}

	// recoverFromErr to true in order to refresh the certificates immediately if we cannot parse them.
	timeToNextRefresh, _ = r.getTimeToRefresh(certificates, true)
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

	log.Infof("successfully refreshed %v", certsDescription)

	// recoverFromErr to so the ticker knows this is an error, and it retries with backoff.
	return r.getTimeToRefresh(certificates, false)
}

func (r *certRefresherImpl) getTimeToRefresh(certificates *storage.TypedServiceCertificateSet, recoverFromErr bool) (time.Duration, error) {
	renewalTime, err := r.getCertsRenewalTime(certificates)
	if err != nil {
		log.Errorf("error getting local scanner certificate expiration, will refresh certificates immediately: %s", err)
		if recoverFromErr {
			return 0, nil
		}
		return 0, err

	}

	return time.Until(renewalTime), nil
}
