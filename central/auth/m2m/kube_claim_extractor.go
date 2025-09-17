package m2m

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
	v1 "k8s.io/api/authentication/v1"
)

type kubeClaimExtractor struct{}

var _ claimExtractor = (*kubeClaimExtractor)(nil)

func (g *kubeClaimExtractor) ExtractRoxClaims(claims map[string][]string) (tokens.RoxClaims, error) {
	if len(claims["sub"]) == 0 {
		return tokens.RoxClaims{}, errox.InvalidArgs.New("no sub claim found")
	}
	if len(claims["uid"]) == 0 {
		return tokens.RoxClaims{}, errox.InvalidArgs.New("no uid claim found")
	}
	sub := claims["sub"][0]
	return tokens.RoxClaims{
		ExternalUser: &tokens.ExternalUserClaim{
			UserID:     claims["uid"][0],
			FullName:   sub,
			Attributes: claims,
		},
		Name: sub,
	}, nil
}

func (g *kubeClaimExtractor) ExtractClaims(idToken *IDToken) (map[string][]string, error) {
	var trs v1.TokenReviewStatus
	if err := idToken.Claims(&trs); err != nil {
		return nil, errors.Wrap(err, "extracting claims")
	}

	claims := map[string][]string{
		"sub":    {trs.User.Username},
		"groups": trs.User.Groups,
		"aud":    trs.Audiences,
		"uid":    {trs.User.UID},
	}
	for k, v := range trs.User.Extra {
		claims[k] = []string(v)
	}
	return claims, nil
}
