package tokens

import (
	"github.com/stackrox/rox/pkg/jwt"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// CreateIssuerFactoryAndValidator is the entrypoint of the auth tokens subsystem, creating an issuer factory that can
// issue token for token sources, as well as a corresponding validator that validates token issued by those issuers.
func CreateIssuerFactoryAndValidator(issuerID string, privateKeyGetter jwt.PrivateKeyStore, publicKeyGetter jwt.PublicKeyGetter, keyID string, options ...Option) (IssuerFactory, Validator) {
	srcs := newSourceStore()
	signerGetter, jwtValidator := jwt.CreateRS256SignerAndValidator(issuerID, nil, privateKeyGetter, publicKeyGetter, keyID)

	factory := newIssuerFactory(issuerID, signerGetter, srcs, options...)
	validator := newValidator(srcs, jwtValidator)
	return factory, validator
}
