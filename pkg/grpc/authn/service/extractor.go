package service

import (
	"context"

	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/mtls"
)

type extractor struct {
	caFP string
}

func (e extractor) IdentityForRequest(_ context.Context, ri requestinfo.RequestInfo) (authn.Identity, error) {
	l := len(ri.VerifiedChains)
	// For all mTLS communication, there will be exactly one verified chain.
	// If there are multiple verified chains, no need to send an error -- it just
	// likely means this is an end user authenticating with client certificates,
	// not an mTLS user.
	if l != 1 {
		return nil, nil
	}

	requestCA := ri.VerifiedChains[0][len(ri.VerifiedChains[0])-1]
	if requestCA.CertFingerprint != e.caFP {
		return nil, nil
	}

	leaf := ri.VerifiedChains[0][0]
	return WrapMTLSIdentity(mtls.IdentityFromCert(leaf)), nil
}

// NewExtractor returns a new identity extractor for internal services.
func NewExtractor() (authn.IdentityExtractor, error) {
	ca, _, err := mtls.CACert()
	if err != nil {
		return nil, err
	}

	caFP := cryptoutils.CertFingerprint(ca)
	return extractor{
		caFP: caFP,
	}, nil
}
