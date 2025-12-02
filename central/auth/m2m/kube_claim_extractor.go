package m2m

import (
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
)

type kubeClaimExtractor struct{}

var _ claimExtractor = (*kubeClaimExtractor)(nil)

func (*kubeClaimExtractor) ExtractRoxClaims(claims map[string][]string) (tokens.RoxClaims, error) {
	if len(claims["sub"]) == 0 {
		return tokens.RoxClaims{}, errox.InvalidArgs.New("no sub claim found")
	}
	if len(claims["kubernetes.io.serviceaccount.uid"]) == 0 {
		return tokens.RoxClaims{}, errox.InvalidArgs.New("no kubernetes.io.serviceaccount.uid claim found")
	}
	sub := claims["sub"][0]
	return tokens.RoxClaims{
		ExternalUser: &tokens.ExternalUserClaim{
			UserID:     claims["kubernetes.io.serviceaccount.uid"][0],
			FullName:   sub,
			Attributes: claims,
		},
		Name: sub,
	}, nil
}

func (*kubeClaimExtractor) ExtractClaims(idToken *IDToken) (map[string][]string, error) {
	return (*genericClaimExtractor)(nil).ExtractClaims(idToken)
}
