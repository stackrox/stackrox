package m2m

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
)

type genericTokenVerifier struct {
	provider *oidc.Provider
	audience string
}

var _ tokenVerifier = (*genericTokenVerifier)(nil)

func (g *genericTokenVerifier) VerifyIDToken(ctx context.Context, rawIDToken string) (*IDToken, error) {
	// When an expected audience is configured, validate the token's aud claim via the go-oidc
	// ClientID check. Otherwise skip it for backward compatibility: a single client ID cannot
	// cover all cases (e.g. GitHub Actions tokens use per-repository audience values), and the
	// audience may still be verified indirectly through claim mappings.
	oidcConfig := &oidc.Config{SkipClientIDCheck: true}
	if g.audience != "" {
		oidcConfig.ClientID = g.audience
		oidcConfig.SkipClientIDCheck = false
	}
	verifier := g.provider.Verifier(oidcConfig)

	token, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}
	return &IDToken{
		Subject:  token.Subject,
		Audience: token.Audience,
		Claims:   token.Claims,
	}, nil
}
