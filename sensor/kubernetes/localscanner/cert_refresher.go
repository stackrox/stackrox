package localscanner

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	log = logging.LoggerForModule()
	certsDescription = "local scanner credentials"
	_ certRefresher = (*certRefresherImpl)(nil)
)

type certRefresher interface {
	Start()
	Stop()
}

func newCertRefresher(requestCertificates requestCertificatesFunc, timeout time.Duration,
	backoff wait.Backoff) certRefresher {
	return &certRefresherImpl{
		requestCertificates: requestCertificates,
		certRefreshTimeout: timeout,
		certRefreshBackoff: backoff,
	}
}

type certRefresherImpl struct {
	requestCertificates requestCertificatesFunc
	certRefreshTimeout time.Duration
	certRefreshBackoff wait.Backoff
	certRefreshTicker  *concurrency.RetryTicker
	certSecretsRepoImpl // FIXME to composition
}

type requestCertificatesFunc func(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error)

func (i *certRefresherImpl) Start() {
	i.certRefreshTicker = concurrency.NewRetryTicker(i.RefreshCertificates, i.certRefreshTimeout, i.certRefreshBackoff)
	i.certRefreshTicker.OnTickSuccess = i.logRefreshSuccess
	i.certRefreshTicker.OnTickError = i.logRefreshError
	i.certRefreshTicker.Start()
}

func (i *certRefresherImpl) Stop() {
	if i.certRefreshTicker != nil {
		i.certRefreshTicker.Stop()
	}
}

// RefreshCertificates determines refreshes the certificate secrets if needed, and returns the time
// until the next refresh.
// This is running in the goroutine for a refresh timer in i.certRefresher.
func (i *certRefresherImpl) RefreshCertificates(ctx context.Context) (timeToRefresh time.Duration, err error) {
	secrets, fetchErr := i.getSecrets(ctx)
	if fetchErr != nil {
		return 0, fetchErr
	}
	timeToRefresh = time.Until(i.getCertRenewalTime(secrets))
	if timeToRefresh > 0 {
		return timeToRefresh, nil
	}

	response, requestErr := i.requestCertificates(ctx)
	if requestErr != nil {
		return 0, requestErr
	}
	if response.GetError() != nil {
		return 0, errors.Errorf("central refused to issue certificates: %s", response.GetError().GetMessage())
	}

	certificates := response.GetCertificates()
	if refreshErr := i.updateSecrets(ctx, certificates, secrets); refreshErr != nil {
		return 0, refreshErr
	}
	timeToRefresh = time.Until(i.getCertRenewalTime(secrets))
	return timeToRefresh, nil
}

func (i *certRefresherImpl) logRefreshSuccess(nextTimeToTick time.Duration) {
	log.Infof("successfully refreshed %v", certsDescription)
	log.Infof("%v scheduled to be refreshed in %s", certsDescription, nextTimeToTick)
}

func (i *certRefresherImpl) logRefreshError(refreshErr error) {
	log.Errorf("refreshing %s: %s", certsDescription, refreshErr)
}

func (i *certRefresherImpl) getCertRenewalTime(secrets map[storage.ServiceType]*v1.Secret) time.Time {
	return time.Now() // TODO
}

// updateSecrets stores the certificates in the data of each corresponding secret, and then persists
// the secrets in k8s.
func (i *certRefresherImpl) updateSecrets(ctx context.Context, certificates *storage.TypedServiceCertificateSet,
	secrets map[storage.ServiceType]*v1.Secret) error {
	// TODO
	return i.putSecrets(ctx, secrets)
}
