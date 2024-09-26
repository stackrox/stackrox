package localscanner

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	certsDescription = "local scanner credentials"
)

// newCertificatesRefresher returns a new retry ticker that uses `requestCertificates` to fetch certificates,
// with the timeout and backoff strategy specified, and the specified repository for persistence.
// Once started, the ticker will periodically refresh the certificates before expiration.
func newCertificatesRefresher(requestCertificates requestCertificatesFunc, repository serviceCertificatesRepo,
	timeout time.Duration, backoff wait.Backoff) concurrency.RetryTicker {

	return concurrency.NewRetryTicker(func(ctx context.Context) (timeToNextTick time.Duration, err error) {
		return refreshCertificates(ctx, requestCertificates, GetCertsRenewalTime, repository)
	}, timeout, backoff)
}

type requestCertificatesFunc func(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error)
type getCertsRenewalTimeFunc func(certificates *storage.TypedServiceCertificateSet) (time.Time, error)

// serviceCertificatesRepo is in charge of persisting and retrieving a set of service certificates, thus implementing
// the [repository pattern](https://martinfowler.com/eaaCatalog/repository.html) for *storage.TypedServiceCertificateSet.
type serviceCertificatesRepo interface {
	// getServiceCertificates retrieves the certificates from permanent storage.
	getServiceCertificates(ctx context.Context) (*storage.TypedServiceCertificateSet, error)
	// ensureServiceCertificates persists the certificates on permanent storage.
	ensureServiceCertificates(ctx context.Context, certificates *storage.TypedServiceCertificateSet) ([]*storage.TypedServiceCertificate, error)
}

// refreshCertificates refreshes the certificate secrets if needed, and returns the time
// until the next refresh.
func refreshCertificates(ctx context.Context, requestCertificates requestCertificatesFunc,
	getCertsRenewalTime getCertsRenewalTimeFunc, repository serviceCertificatesRepo) (timeToNextRefresh time.Duration, err error) {

	timeToNextRefresh, err = ensureCertificatesAreFresh(ctx, requestCertificates, getCertsRenewalTime, repository)
	if err != nil {
		if errors.Is(err, ErrUnexpectedSecretsOwner) {
			log.Errorf("non-recoverable error refreshing %s, automatic refresh will be stopped: %s", certsDescription, err)
			return 0, concurrency.ErrNonRecoverable
		}

		log.Errorf("refreshing %s: %s", certsDescription, err)
		return 0, err
	}

	log.Infof("%v scheduled to be refreshed in %s", certsDescription, timeToNextRefresh)
	return timeToNextRefresh, err
}

func ensureCertificatesAreFresh(ctx context.Context, requestCertificates requestCertificatesFunc,
	getCertsRenewalTime getCertsRenewalTimeFunc, repository serviceCertificatesRepo) (time.Duration, error) {

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

	persistedCertificates, putErr := repository.ensureServiceCertificates(ctx, certificates)
	if putErr != nil {
		return 0, putErr
	}

	renewalTime, err := getCertsRenewalTime(certificates)
	if err != nil {
		// send the error to the ticker, so it retries with backoff.
		return 0, err
	}
	serviceTypeNames := getServiceTypeNames(persistedCertificates)
	log.Infof("successfully refreshed %v for: %v", certsDescription, strings.Join(serviceTypeNames, ", "))
	return time.Until(renewalTime), nil
}

func getServiceTypeNames(serviceCertificates []*storage.TypedServiceCertificate) []string {
	serviceTypeNames := make([]string, 0, len(serviceCertificates))
	for _, c := range serviceCertificates {
		serviceTypeNames = append(serviceTypeNames, c.ServiceType.String())
	}
	return serviceTypeNames
}

func getTimeToRefreshFromRepo(ctx context.Context, getCertsRenewalTime getCertsRenewalTimeFunc,
	repository serviceCertificatesRepo) (time.Duration, error) {

	certificates, getCertsErr := repository.getServiceCertificates(ctx)
	if errors.Is(getCertsErr, ErrUnexpectedSecretsOwner) {
		return 0, getCertsErr
	}
	if errors.Is(getCertsErr, ErrDifferentCAForDifferentServiceTypes) || errors.Is(getCertsErr, ErrMissingSecretData) {
		log.Errorf("local scanner certificates are in an inconsistent state, "+
			"will refresh certificates immediately: %s", getCertsErr)
		return 0, nil
	}
	if k8sErrors.IsNotFound(getCertsErr) {
		log.Warnf("local scanner certificates not found (this is expected on a new deployment), "+
			"will refresh certificates immediately: %s", getCertsErr)
		return 0, nil
	}
	if getCertsErr != nil {
		return 0, getCertsErr
	}

	renewalTime, err := getCertsRenewalTime(certificates)
	if err != nil {
		// recover by refreshing the certificates immediately.
		log.Errorf("error getting local scanner certificate expiration, "+
			"will refresh certificates immediately: %s", err)
		return 0, nil
	}
	return time.Until(renewalTime), nil
}
