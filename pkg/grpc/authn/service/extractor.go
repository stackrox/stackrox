package service

import (
	"context"
	"errors"

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
	if l == 0 {
		return nil, nil
	}
	if l != 1 {
		return nil, errors.New("client presented multiple certificates; this is unsupported")
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
