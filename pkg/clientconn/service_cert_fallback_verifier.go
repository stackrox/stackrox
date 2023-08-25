package clientconn

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/tlscheck"
)

type serviceCertFallbackVerifier struct {
	serviceCAs *x509.CertPool
	subject    mtls.Subject
}

func isServiceCert(cert *x509.Certificate, subj mtls.Subject) bool {
	if cert.Issuer.CommonName != mtls.ServiceCACommonName {
		return false
	}
	if cert.Subject.CommonName == subj.CN() {
		return true
	}
	if len(cert.Subject.OrganizationalUnit) != 1 {
		return false
	}
	return cert.Subject.OrganizationalUnit[0] == subj.OU()
}

func (v *serviceCertFallbackVerifier) VerifyPeerCertificate(leaf *x509.Certificate, chainRest []*x509.Certificate, conf *tls.Config) error {
	intermediates := tlscheck.NewCertPool(chainRest...)

	systemVerifyOpts := x509.VerifyOptions{
		DNSName:       conf.ServerName,
		Intermediates: intermediates,
		Roots:         conf.RootCAs,
	}

	_, systemVerifyErr := leaf.Verify(systemVerifyOpts)
	if systemVerifyErr == nil || !isServiceCert(leaf, v.subject) {
		return systemVerifyErr
	}

	verifyErrs := errorhelpers.NewErrorList(fmt.Sprintf("verifying %s certificate", v.subject.Identifier))
	verifyErrs.AddError(systemVerifyErr)

	serviceVerifyOpts := x509.VerifyOptions{
		DNSName:       v.subject.Hostname(),
		Intermediates: intermediates,
		Roots:         v.serviceCAs,
	}

	_, serviceVerifyErr := leaf.Verify(serviceVerifyOpts)
	if serviceVerifyErr == nil {
		return nil
	}
	verifyErrs.AddError(serviceVerifyErr)
	return verifyErrs.ToError()
}
