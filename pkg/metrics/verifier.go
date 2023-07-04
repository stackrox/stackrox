package metrics

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
)

type clientCertVerifier struct {
	subjectCN string
}

func (v *clientCertVerifier) VerifyPeerCertificate(leaf *x509.Certificate, chainRest []*x509.Certificate, tlsConfig *tls.Config) error {
	verifyErrs := errorhelpers.NewErrorList("verifying certificate")
	if leaf.Subject.CommonName != v.subjectCN {
		verifyErrs.AddError(errox.NotAuthorized.CausedByf("expected Subject.CN=%q, got %q", v.subjectCN, leaf.Subject.CommonName))
	}

	intermediates := clientconn.NewCertPool(chainRest...)
	clientVerifyOpts := x509.VerifyOptions{
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Roots:         tlsConfig.ClientCAs,
	}
	_, clientVerifyErr := leaf.Verify(clientVerifyOpts)
	if clientVerifyErr != nil {
		verifyErrs.AddError(errox.NotAuthorized.CausedBy(clientVerifyErr))
	}

	return verifyErrs.ToError()
}
