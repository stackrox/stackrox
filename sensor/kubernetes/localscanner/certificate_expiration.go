package localscanner

import (
	"time"

	"crypto/x509"
	"math/rand"
	"github.com/cloudflare/cfssl/helpers"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mtls"
	v1 "k8s.io/api/core/v1"
)

// GetSecretsCertRenewalTime computes the time when the service certificates stored in a set of
// secrets should be refreshed.
// If different services have different expiration times then the earliest time is returned.
func GetSecretsCertRenewalTime(secrets map[storage.ServiceType]*v1.Secret) (time.Time, error) {
	var (
		renewalTime            time.Time
		renewalTimeInitialized bool
	)
	for _, secret := range secrets {
		secretRenewalTime, err := getSecretRenewalTime(secret)
		if err != nil {
			return renewalTime, err
		}
		if !renewalTimeInitialized || secretRenewalTime.Before(renewalTime) {
			renewalTimeInitialized = true
			renewalTime = secretRenewalTime
		}
	}
	return renewalTime, nil
}

func getSecretRenewalTime(secret *v1.Secret) (time.Time, error) {
	certBytes := secret.Data[mtls.ServiceCertFileName]
	var (
		scannerCert *x509.Certificate
		err         error
	)
	if len(certBytes) == 0 {
		err = errors.Errorf("empty certificate for secret %s", secret.GetName())
	} else {
		scannerCert, err = helpers.ParseCertificatePEM(certBytes)
	}
	if err != nil {
		// Note this also covers a secret with no certificates stored, which should be refreshed immediately.
		return time.Now(), err
	}

	return getSecretRenewalTimeFromCertificate(scannerCert), nil
}

func getSecretRenewalTimeFromCertificate(certificate *x509.Certificate) time.Time {
	certValidityDurationSecs := certificate.NotAfter.Sub(certificate.NotBefore).Seconds()
	durationBeforeRenewalAttempt := time.Second *
		(time.Duration(certValidityDurationSecs/2) - time.Duration(rand.Intn(int(certValidityDurationSecs/10))))
	certRenewalTime := certificate.NotBefore.Add(durationBeforeRenewalAttempt)
	return certRenewalTime
}
