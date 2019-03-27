package saml

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	saml2 "github.com/russellhaering/gosaml2"
	dsig "github.com/russellhaering/goxmldsig"
)

func configureIDPFromSettings(sp *saml2.SAMLServiceProvider, idpIssuer, idpLoginURL, idpCertPEM string) error {
	sp.IdentityProviderIssuer = idpIssuer
	sp.IdentityProviderSSOURL = idpLoginURL

	certStore := &dsig.MemoryX509CertificateStore{}
	certDERBlock, rest := pem.Decode([]byte(idpCertPEM))
	if certDERBlock == nil || certDERBlock.Type != "CERTIFICATE" {
		return errors.New("PEM data does not look like a certificate")
	}
	if len(rest) != 0 {
		return fmt.Errorf("%d extra bytes in PEM data", len(rest))
	}
	parsedCert, err := x509.ParseCertificate(certDERBlock.Bytes)
	if err != nil {
		return fmt.Errorf("could not parse certificate: %v", err)
	}
	certStore.Roots = append(certStore.Roots, parsedCert)
	sp.IDPCertificateStore = certStore

	return nil
}
