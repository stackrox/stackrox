package saml

import (
	"github.com/pkg/errors"
	saml2 "github.com/russellhaering/gosaml2"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/stackrox/rox/pkg/x509utils"
)

func configureIDPFromSettings(sp *saml2.SAMLServiceProvider, idpIssuer, idpLoginURL, idpCertPEM, nameIDFormat string) error {
	sp.IdentityProviderIssuer = idpIssuer
	sp.IdentityProviderSSOURL = idpLoginURL
	sp.NameIdFormat = nameIDFormat

	certs, err := x509utils.ParseCertificatesPEM([]byte(idpCertPEM))
	if err != nil {
		return errors.Wrap(err, "parsing certificate PEM data")
	}

	sp.IDPCertificateStore = &dsig.MemoryX509CertificateStore{Roots: certs}

	return nil
}
