package metrics

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/hashicorp/go-multierror"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/tlscheck"
)

type clientCertVerifier struct {
	subjectCN string
}

func (v *clientCertVerifier) VerifyPeerCertificate(leaf *x509.Certificate, chainRest []*x509.Certificate, tlsConfig *tls.Config) error {
	var verifyErr error
	if leaf.Subject.CommonName != v.subjectCN {
		noAuthErr := errox.NotAuthorized.CausedByf("expected Subject.CN=%q, got %q", v.subjectCN, leaf.Subject.CommonName)
		verifyErr = multierror.Append(verifyErr, noAuthErr)
	}

	intermediates := tlscheck.NewCertPool(chainRest...)
	clientVerifyOpts := x509.VerifyOptions{
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Roots:         tlsConfig.ClientCAs,
	}
	_, clientVerifyErr := leaf.Verify(clientVerifyOpts)
	if clientVerifyErr != nil {
		verifyErr = multierror.Append(verifyErr, errox.NotAuthorized.CausedBy(clientVerifyErr))
	}

	return verifyErr
}
