package service

import (
	"context"
	"errors"

	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/mtls"
)

type extractor struct{}

func (extractor) IdentityForRequest(_ context.Context, ri requestinfo.RequestInfo) (authn.Identity, error) {
	l := len(ri.VerifiedChains)
	if l == 0 {
		return nil, nil
	}
	if l != 1 {
		return nil, errors.New("client presented multiple certificates; this is unsupported")
	}
	ca, _, err := mtls.CACert()

	if err != nil {
		return nil, err
	}

	fingerprint := cryptoutils.CertFingerprint(ca)
	requestCA := ri.VerifiedChains[0][len(ri.VerifiedChains[0])-1]
	if fingerprint != requestCA.CertFingerprint {
		return nil, nil
	}

	leaf := ri.VerifiedChains[0][0]
	return identity{id: mtls.IdentityFromCert(leaf)}, nil
}

// NewExtractor returns a new identity extractor for internal services.
func NewExtractor() authn.IdentityExtractor {
	return extractor{}
}
