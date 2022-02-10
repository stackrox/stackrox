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
	// ErrDifferentCAForDifferentServiceTypes TODO replace by ROX-9128
	ErrDifferentCAForDifferentServiceTypes = errors.New("found different CA PEM in secret Data for different service types")
	// ErrMissingSecretData TODO replace by ROX-9128
	ErrMissingSecretData = errors.New("missing secret data")
)

// newCertificatesRefresher returns a new retry ticker that uses `requestCertificates` to fetch certificates,
// with the timeout and backoff strategy specified, and the specified repository for persistence.
// Once started, the ticker will periodically refresh the certificates before expiration.
func newCertificatesRefresher(requestCertificates requestCertificatesFunc, repository ServiceCertificatesRepo,
	timeout time.Duration, backoff wait.Backoff) concurrency.RetryTicker {
	return concurrency.NewRetryTicker(func(ctx context.Context) (timeToNextTick time.Duration, err error) {
		return refreshCertificates(ctx, requestCertificates, GetCertsRenewalTime, repository)
	}, timeout, backoff)
}

type requestCertificatesFunc func(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error)
type getCertsRenewalTimeFunc func(certificates *storage.TypedServiceCertificateSet) (time.Time, error)

// ServiceCertificatesRepo is in charge of persisting and retrieving a set of service certificates, thus implementing
// the [repository pattern](https://martinfowler.com/eaaCatalog/repository.html) for *storage.TypedServiceCertificateSet.
type ServiceCertificatesRepo interface {
	// GetServiceCertificates retrieves the certificates from permanent storage.
	GetServiceCertificates(ctx context.Context) (*storage.TypedServiceCertificateSet, error)
	// PutServiceCertificates persists the certificates on permanent storage.
	PutServiceCertificates(ctx context.Context, certificates *storage.TypedServiceCertificateSet) error
}

// refreshCertificates determines refreshes the certificate secrets if needed, and returns the time
// until the next refresh.
func refreshCertificates(ctx context.Context,
	requestCertificates requestCertificatesFunc,
	getCertsRenewalTime getCertsRenewalTimeFunc,
	repository ServiceCertificatesRepo) (timeToNextRefresh time.Duration, err error) {

	timeToNextRefresh, err = ensureCertificatesAreFresh(ctx, requestCertificates, getCertsRenewalTime, repository)
	if err != nil {
		log.Errorf("refreshing %s: %s", certsDescription, err)
		return 0, err
	}

	log.Infof("%v scheduled to be refreshed in %s", certsDescription, timeToNextRefresh)
	return timeToNextRefresh, err
}

func ensureCertificatesAreFresh(ctx context.Context, requestCertificates requestCertificatesFunc,
	getCertsRenewalTime getCertsRenewalTimeFunc, repository ServiceCertificatesRepo) (time.Duration, error) {

	timeToRefresh, getCertsErr := getTimeToRefreshFromRepo(ctx, getCertsRenewalTime, repository)
	if getCertsErr != nil {
		return 0, getCertsErr
	}

	if timeToRefresh > 0 {
		return timeToRefresh, nil
	}

	response, requestErr := requestCertificates(ctx)
	if requestErr != nil {
		return 0, requestErr
	}
	if response.GetError() != nil {
		return 0, errors.Errorf("central refused to issue certificates: %s", response.GetError().GetMessage())
	}
	certificates := response.GetCertificates()

	if putErr := repository.PutServiceCertificates(ctx, certificates); putErr != nil {
		return 0, putErr
	}

	// recoverFromErr to so the ticker knows this is an error, and it retries with backoff.
	timeToNextRefresh, err := getTimeToRefreshFromCertificates(getCertsRenewalTime, certificates, false)
	if err == nil {
		log.Infof("successfully refreshed %v", certsDescription)
	}

	return timeToNextRefresh, err
}

func getTimeToRefreshFromRepo(ctx context.Context, getCertsRenewalTime getCertsRenewalTimeFunc,
	repository ServiceCertificatesRepo) (time.Duration, error) {

	certificates, getCertsErr := repository.GetServiceCertificates(ctx)
	if getCertsErr == ErrDifferentCAForDifferentServiceTypes || getCertsErr == ErrMissingSecretData {
		log.Errorf("local scanner certificates are corrupted, will refresh certificates immediately: %s", getCertsErr)
		return 0, nil
	}
	if getCertsErr != nil {
		return 0, getCertsErr
	}

	// recoverFromErr to true in order to refresh the certificates immediately if we cannot parse them.
	return getTimeToRefreshFromCertificates(getCertsRenewalTime, certificates, true)
}

func getTimeToRefreshFromCertificates(getCertsRenewalTime getCertsRenewalTimeFunc,
	certificates *storage.TypedServiceCertificateSet, recoverFromErr bool) (time.Duration, error) {

	renewalTime, err := getCertsRenewalTime(certificates)
	if err != nil {
		if recoverFromErr {
			log.Errorf("error getting local scanner certificate expiration, will refresh certificates immediately: %s", err)
			return 0, nil
		}
		return 0, err
	}

	return time.Until(renewalTime), nil
}
