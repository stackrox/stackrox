package localscanner

import (
	"crypto/x509"
	"math/rand"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

var (
	// ErrEmptyCertificate indicates that the certificate stored in a secret is empty.
	ErrEmptyCertificate = errors.New("empty certificate")
)

// GetCertsRenewalTime computes the time when the service certificates should be refreshed.
// If different services have different expiration times then the earliest time is returned.
func GetCertsRenewalTime(certificates *storage.TypedServiceCertificateSet) (time.Time, error) {
	var (
		renewalTime            time.Time
		renewalTimeInitialized bool
	)
	for _, certificate := range certificates.GetServiceCerts() {
		certRenewalTime, err := getCertificateRenewalTime(certificate)
		if err != nil {
			return renewalTime, err
		}
		if !renewalTimeInitialized || certRenewalTime.Before(renewalTime) {
			renewalTimeInitialized = true
			renewalTime = certRenewalTime
		}
	}
	return renewalTime, nil
}

func getCertificateRenewalTime(certificate *storage.TypedServiceCertificate) (time.Time, error) {
	certBytes := certificate.GetCert().GetCertPem()
	var (
		cert *x509.Certificate
		err  error
	)
	if len(certBytes) == 0 {
		err = ErrEmptyCertificate
	} else {
		cert, err = helpers.ParseCertificatePEM(certBytes)
	}
	if err != nil {
		var zeroTime time.Time
		return zeroTime, err
	}

	return calculateRenewalTime(cert), nil
}

// In order to ensure certificates are rotated before expiration, this returns a renewal time no later than
// half its expiration date.
func calculateRenewalTime(cert *x509.Certificate) time.Time {
	certValidityDurationSecs := cert.NotAfter.Sub(cert.NotBefore).Seconds()
	durationBeforeRenewalAttempt := time.Second *
		(time.Duration(certValidityDurationSecs/2) - time.Duration(rand.Intn(int(certValidityDurationSecs/10))))
	certRenewalTime := cert.NotBefore.Add(durationBeforeRenewalAttempt)
	return certRenewalTime
}
