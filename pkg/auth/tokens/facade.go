package tokens

import (
	"crypto/rsa"

	"github.com/stackrox/stackrox/pkg/jwt"
	"github.com/stackrox/stackrox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// CreateIssuerFactoryAndValidator is the entrypoint of the auth tokens subsystem, creating an issuer factory that can
// issue token for token sources, as well as a corresponding validator that validates token issued by those issuers.
func CreateIssuerFactoryAndValidator(issuerID string, privateKey *rsa.PrivateKey, keyID string, options ...Option) (IssuerFactory, Validator, error) {
	srcs := newSourceStore()
	signer, jwtValidator, err := jwt.CreateRS256SignerAndValidator(issuerID, nil, privateKey, keyID)
	if err != nil {
		return nil, nil, err
	}

	factory := newIssuerFactory(issuerID, signer, srcs, options...)
	validator := newValidator(srcs, jwtValidator)
	return factory, validator, nil
}
