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

func getSecretRenewalTime(certSecret *v1.Secret) (time.Time, error) {
	certBytes := certSecret.Data[mtls.ServiceCertFileName]
	var (
		cert *x509.Certificate
		err         error
	)
	if len(certBytes) == 0 {
		err = errors.Errorf("empty certificate for certSecret %s", certSecret.GetName())
	} else {
		cert, err = helpers.ParseCertificatePEM(certBytes)
	}
	if err != nil {
		// Note this also covers a certSecret with no certificates stored, which should be refreshed immediately.
		return time.Now(), err
	}

	return calculateRenewalTime(cert), nil
}

func calculateRenewalTime(cert *x509.Certificate) time.Time {
	certValidityDurationSecs := cert.NotAfter.Sub(cert.NotBefore).Seconds()
	durationBeforeRenewalAttempt := time.Second *
		(time.Duration(certValidityDurationSecs/2) - time.Duration(rand.Intn(int(certValidityDurationSecs/10))))
	certRenewalTime := cert.NotBefore.Add(durationBeforeRenewalAttempt)
	return certRenewalTime
}
