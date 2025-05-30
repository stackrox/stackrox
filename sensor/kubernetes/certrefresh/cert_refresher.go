package certrefresh

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/kubernetes/certrefresh/certrepo"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

// newCertificatesRefresher returns a new retry ticker that uses `requestCertificates` to fetch certificates,
// with the timeout and backoff strategy specified, and the specified repository for persistence.
// Once started, the ticker will periodically refresh the certificates before expiration.
func newCertificatesRefresher(certsDescription string, requestCertificates requestCertificatesFunc, repository certrepo.ServiceCertificatesRepo,
	timeout time.Duration, backoff wait.Backoff) concurrency.RetryTicker {

	return concurrency.NewRetryTicker(func(ctx context.Context) (timeToNextTick time.Duration, err error) {
		return refreshCertificates(ctx, certsDescription, requestCertificates, GetCertsRenewalTime, repository)
	}, timeout, backoff)
}

type requestCertificatesFunc func(ctx context.Context) (*Response, error)
type getCertsRenewalTimeFunc func(certificates *storage.TypedServiceCertificateSet) (time.Time, error)

// refreshCertificates refreshes the certificate secrets if needed, and returns the time
// until the next refresh.
func refreshCertificates(ctx context.Context, certsDescription string, requestCertificates requestCertificatesFunc,
	getCertsRenewalTime getCertsRenewalTimeFunc, repository certrepo.ServiceCertificatesRepo) (timeToNextRefresh time.Duration, err error) {

	timeToNextRefresh, err = ensureCertificatesAreFresh(ctx, certsDescription, requestCertificates, getCertsRenewalTime, repository)
	if err != nil {
		if errors.Is(err, certrepo.ErrUnexpectedSecretsOwner) {
			log.Errorf("non-recoverable error refreshing %s TLS certificates, automatic refresh will be stopped: %s", certsDescription, err)
			return 0, concurrency.ErrNonRecoverable
		}

		log.Errorf("refreshing %s TLS certificates: %s", certsDescription, err)
		return 0, err
	}

	log.Infof("%v TLS certificates scheduled to be refreshed in %s", certsDescription, timeToNextRefresh)
	return timeToNextRefresh, err
}

func ensureCertificatesAreFresh(ctx context.Context, certsDescription string, requestCertificates requestCertificatesFunc,
	getCertsRenewalTime getCertsRenewalTimeFunc, repository certrepo.ServiceCertificatesRepo) (time.Duration, error) {

	timeToRefresh, getCertsErr := getTimeToRefreshFromRepo(ctx, certsDescription, getCertsRenewalTime, repository)
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
	if response.ErrorMessage != nil {
		return 0, errors.Errorf("central refused to issue certificates: %s", *response.ErrorMessage)
	}
	certificates := response.Certificates
	if certificates == nil {
		return 0, errors.New("certificates set is nil")
	}

	persistedCertificates, putErr := repository.EnsureServiceCertificates(ctx, certificates)
	if putErr != nil {
		return 0, errors.Wrap(putErr, "saving certificates to repository")
	}

	renewalTime, err := getCertsRenewalTime(certificates)
	if err != nil {
		// send the error to the ticker, so it retries with backoff.
		return 0, err
	}
	serviceTypeNames := getServiceTypeNames(persistedCertificates)
	log.Infof("successfully refreshed %v TLS certificates for: %v", certsDescription, strings.Join(serviceTypeNames, ", "))
	return time.Until(renewalTime), nil
}

func getServiceTypeNames(serviceCertificates []*storage.TypedServiceCertificate) []string {
	serviceTypeNames := make([]string, 0, len(serviceCertificates))
	for _, c := range serviceCertificates {
		serviceTypeNames = append(serviceTypeNames, c.ServiceType.String())
	}
	return serviceTypeNames
}

func getTimeToRefreshFromRepo(ctx context.Context, certsDescription string, getCertsRenewalTime getCertsRenewalTimeFunc,
	repository certrepo.ServiceCertificatesRepo) (time.Duration, error) {

	certificates, getCertsErr := repository.GetServiceCertificates(ctx)
	if errors.Is(getCertsErr, certrepo.ErrUnexpectedSecretsOwner) {
		return 0, errors.Wrapf(getCertsErr, "getting %s certificates from repository", certsDescription)
	}
	if errors.Is(getCertsErr, certrepo.ErrDifferentCAForDifferentServiceTypes) || errors.Is(getCertsErr, certrepo.ErrMissingSecretData) {
		log.Errorf("%s TLS certificates are in an inconsistent state, "+
			"will refresh certificates immediately: %s", certsDescription, getCertsErr)
		return 0, nil
	}
	if k8sErrors.IsNotFound(getCertsErr) {
		log.Warnf("%s TLS certificates not found (this is expected on a new deployment), "+
			"will refresh certificates immediately: %s", certsDescription, getCertsErr)
		return 0, nil
	}
	if getCertsErr != nil {
		return 0, errors.Wrapf(getCertsErr, "getting %s certificates from repository", certsDescription)
	}

	renewalTime, err := getCertsRenewalTime(certificates)
	if err != nil {
		// recover by refreshing the certificates immediately.
		log.Errorf("error getting %s TLS certificates expiration, "+
			"will refresh certificates immediately: %s", certsDescription, err)
		return 0, nil
	}
	return time.Until(renewalTime), nil
}
