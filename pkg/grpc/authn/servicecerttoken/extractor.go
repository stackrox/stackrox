package servicecerttoken

import (
	"context"
	"crypto/x509"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/service"
	"github.com/stackrox/rox/pkg/grpc/common/authn/servicecerttoken"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
)

var (
	log = logging.LoggerForModule()
)

type extractor struct {
	verifyOpts x509.VerifyOptions
	maxLeeway  time.Duration
	validator  authn.ValidateCertChain
}

func (e extractor) IdentityForRequest(ctx context.Context, ri requestinfo.RequestInfo) (authn.Identity, error) {
	token := authn.ExtractToken(ri.Metadata, servicecerttoken.TokenType)
	if token == "" {
		return nil, nil
	}

	cert, err := servicecerttoken.ParseToken(token, e.maxLeeway)
	if err != nil {
		log.Warnf("Could not parse service cert token: %v", err)
		return nil, errors.New("could not parse service cert token")
	}

	verifiedChains, err := cert.Verify(e.verifyOpts)
	if err != nil {
		return nil, errors.Wrap(err, "could not verify certificate")
	}

	if len(verifiedChains) != 1 {
		return nil, errors.Errorf("UNEXPECTED: %d verified chains found", len(verifiedChains))
	}

	if len(verifiedChains[0]) == 0 {
		return nil, errors.New("UNEXPECTED: verified chain is empty")
	}

	chain := requestinfo.ExtractCertInfoChains(verifiedChains)
	if e.validator != nil {
		if err := e.validator.ValidateClientCertificate(ctx, chain[0]); err != nil {
			log.Errorf("Could not validate client certificate from service cert token: %v", err)
			return nil, errors.New("could not validate client certificate from service cert token")
		}
	}

	log.Debugf("Woot! Someone (%s) is authenticating with a service cert token", verifiedChains[0][0].Subject)

	return service.WrapMTLSIdentity(mtls.IdentityFromCert(chain[0][0])), nil
}

// NewExtractorWithCertValidation returns an extractor which allows to configure a cert chain validation
func NewExtractorWithCertValidation(maxLeeway time.Duration, validator authn.ValidateCertChain) (authn.IdentityExtractor, error) {
	ca, _, err := mtls.CACert()
	if err != nil {
		return nil, err
	}
	trustPool := x509.NewCertPool()
	trustPool.AddCert(ca)

	verifyOpts := x509.VerifyOptions{
		Roots: trustPool,
	}

	return extractor{
		verifyOpts: verifyOpts,
		maxLeeway:  maxLeeway,
		validator:  validator,
	}, nil
}
