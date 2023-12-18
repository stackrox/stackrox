package servicecerttoken

import (
	"context"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/service"
	"github.com/stackrox/rox/pkg/grpc/common/authn/servicecerttoken"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
)

type extractor struct {
	verifyOpts x509.VerifyOptions
	maxLeeway  time.Duration
	validator  authn.ValidateCertChain
}

func getExtractorError(msg string, err error) *authn.ExtractorError {
	return authn.NewExtractorError("service-cert-token", msg, err)
}

func (e extractor) IdentityForRequest(ctx context.Context, ri requestinfo.RequestInfo) (authn.Identity, *authn.ExtractorError) {
	token := authn.ExtractToken(ri.Metadata, servicecerttoken.TokenType)
	if token == "" {
		return nil, nil
	}

	cert, err := servicecerttoken.ParseToken(token, e.maxLeeway)
	if err != nil {
		return nil, getExtractorError("could not parse service cert token", err)
	}

	verifiedChains, err := cert.Verify(e.verifyOpts)
	if err != nil {
		return nil, getExtractorError("could not verify certificate", err)
	}

	if len(verifiedChains) != 1 {
		return nil, getExtractorError(fmt.Sprintf("UNEXPECTED: %d verified chains found", len(verifiedChains)), nil)
	}

	if len(verifiedChains[0]) == 0 {
		return nil, getExtractorError("UNEXPECTED: verified chain is empty", nil)
	}

	chain := requestinfo.ExtractCertInfoChains(verifiedChains)
	if e.validator != nil {
		if err := e.validator.ValidateClientCertificate(ctx, chain[0]); err != nil {
			return nil, getExtractorError("could not validate client certificate from service cert token", err)
		}
	}

	logging.GetRateLimitedLogger().DebugL(ri.Hostname, "%q is authenticating with a service cert token", verifiedChains[0][0].Subject)

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
