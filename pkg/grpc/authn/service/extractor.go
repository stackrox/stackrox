package service

import (
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/mtls"
)

type extractor struct{}

func (extractor) IdentityForRequest(ri requestinfo.RequestInfo) authn.Identity {
	l := len(ri.VerifiedChains)
	if l != 1 {
		return nil
	}
	leaf := ri.VerifiedChains[0][0]
	return identity{id: mtls.IdentityFromCert(leaf)}
}

// NewExtractor returns a new identity extractor for internal services.
func NewExtractor() authn.IdentityExtractor {
	return extractor{}
}
