package service

import (
	"errors"

	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/mtls"
)

type extractor struct{}

func (extractor) IdentityForRequest(ri requestinfo.RequestInfo) (authn.Identity, error) {
	l := len(ri.VerifiedChains)
	if l == 0 {
		return nil, nil
	}
	if l != 1 {
		return nil, errors.New("client presented multiple certificates; this is unsupported")
	}
	leaf := ri.VerifiedChains[0][0]
	return identity{id: mtls.IdentityFromCert(leaf)}, nil
}

// NewExtractor returns a new identity extractor for internal services.
func NewExtractor() authn.IdentityExtractor {
	return extractor{}
}
