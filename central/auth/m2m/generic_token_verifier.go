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
	// Skip the client id check unless the user has configured the expected audience claim.
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
