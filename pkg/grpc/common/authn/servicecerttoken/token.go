package servicecerttoken

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	ctTLS "github.com/google/certificate-transparency-go/tls"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/protocompat"
)

// ParseToken parses a ServiceCert token and returns the parsed x509 certificate. Note that the returned certificate is
// not verified.
func ParseToken(token string, maxLeeway time.Duration) (*x509.Certificate, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, errors.Errorf("expected token to contain exactly one '.', found %d", len(parts)-1)
	}

	authBytes, err := b64Enc.DecodeString(parts[0])
	if err != nil {
		return nil, errors.Wrap(err, "could not decode auth info")
	}
	sigBytes, err := b64Enc.DecodeString(parts[1])
	if err != nil {
		return nil, errors.Wrap(err, "could not decode signature")
	}

	var auth central.ServiceCertAuth
	if err := auth.UnmarshalVTUnsafe(authBytes); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal service cert auth structure")
	}

	ts, err := protocompat.ConvertTimestampToTimeOrError(auth.GetCurrentTime())
	if err != nil {
		return nil, errors.Wrap(err, "could not convert timestamp")
	}
	tsDiff := time.Since(ts)
	if tsDiff < 0 {
		tsDiff = -tsDiff
	}
	if tsDiff > maxLeeway {
		return nil, errors.Errorf("time difference %v > %v detected", tsDiff, maxLeeway)
	}

	cert, err := x509.ParseCertificate(auth.GetCertDer())
	if err != nil {
		return nil, errors.Wrap(err, "could not parse certificate data")
	}

	ds := ctTLS.DigitallySigned{
		Algorithm: ctTLS.SignatureAndHashAlgorithm{
			Hash:      hashAlgo,
			Signature: ctTLS.SignatureAlgorithmFromPubKey(cert.PublicKey),
		},
		Signature: sigBytes,
	}

	if err := ctTLS.VerifySignature(cert.PublicKey, authBytes, ds); err != nil {
		return nil, errors.Wrap(err, "failed to verify signature")
	}
	return cert, nil
}

// CreateToken creates a ServiceCert token from the given certificate, stamping it with the given current timestamp.
func CreateToken(cert *tls.Certificate, currTime time.Time) (string, error) {
	tsPb, err := protocompat.ConvertTimeToTimestampOrError(currTime)
	if err != nil {
		return "", errors.Wrap(err, "could not create timestamp proto")
	}

	auth := &central.ServiceCertAuth{
		CertDer:     cert.Certificate[0],
		CurrentTime: tsPb,
	}

	authBytes, err := auth.MarshalVT()
	if err != nil {
		return "", errors.Wrap(err, "could not marshal service cert auth structure")
	}

	ds, err := ctTLS.CreateSignature(cryptoutils.DerefPrivateKey(cert.PrivateKey), hashAlgo, authBytes)
	if err != nil {
		return "", errors.Wrap(err, "could not create signature")
	}

	return fmt.Sprintf("%s.%s", b64Enc.EncodeToString(authBytes), b64Enc.EncodeToString(ds.Signature)), nil
}
