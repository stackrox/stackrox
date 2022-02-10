package centralclient

import (
	"crypto/x509"

	cTLS "github.com/google/certificate-transparency-go/tls"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/x509utils"
)

func verifyCentralCertificateChain(x509CertChain []*x509.Certificate, rootCAs *x509.CertPool) error {
	err := x509utils.VerifyCertificateChain(x509CertChain, x509.VerifyOptions{
		Roots:   rootCAs,
		DNSName: mtls.CentralSubject.Hostname(),
	})

	if err != nil {
		return newMismatchCentralErr(err.Error())
	}

	return nil
}

func verifySignatureAgainstCertificate(cert *x509.Certificate, payload []byte, signature []byte) error {
	err := cTLS.VerifySignature(cert.PublicKey, payload, cTLS.DigitallySigned{
		Signature: signature,
		Algorithm: cTLS.SignatureAndHashAlgorithm{
			Hash:      cTLS.SHA256,
			Signature: cTLS.SignatureAlgorithmFromPubKey(cert.PublicKey),
		},
	})

	if err != nil {
		return newTrustInfoSignatureErr(err.Error())
	}
	return nil
}
