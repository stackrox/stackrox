package m2m

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
)

type genericTokenVerifier struct {
	provider *oidc.Provider
}

var _ tokenVerifier = (*genericTokenVerifier)(nil)

func (g *genericTokenVerifier) VerifyIDToken(ctx context.Context, rawIDToken string) (*IDToken, error) {
	verifier := g.provider.Verifier(&oidc.Config{
		// We currently provide no config to expose the client ID that's associated with the ID token.
		// The reason for this is the following:
		// - A magnitude of client IDs would have to be configured (i.e. in the case of GitHub actions, this would be
		// all repository URLs including their potential customizations).
		// - Client IDs (i.e. the "sub" claim) _may_ be part for the mappings within the
		// config. So, essentially the client ID check is deferred to a latter point, as the mappings _may_ be used
		// for mapping, but it currently isn't a requirement.
		SkipClientIDCheck: true,
	})

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
