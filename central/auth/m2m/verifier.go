package m2m

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

var (
	_ tokenVerifier = (*genericTokenVerifier)(nil)
)

type tokenVerifier interface {
	VerifyIDToken(ctx context.Context, rawIDToken string) (*oidc.IDToken, error)
}

func tokenVerifierFromConfig(ctx context.Context, config *storage.AuthMachineToMachineConfig) (tokenVerifier, error) {
	var issuer string
	switch config.GetType() {
	case storage.AuthMachineToMachineConfig_GITHUB_ACTIONS:
		// Ref: https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect#understanding-the-oidc-token.
		issuer = "https://token.actions.githubusercontent.com"
	default:
		issuer = config.GetGenericIssuerConfig().GetIssuer()
	}

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, errors.Wrapf(err, "creating OIDC provider for issuer %q", issuer)
	}

	return &genericTokenVerifier{issuer: issuer, provider: provider}, nil
}

type genericTokenVerifier struct {
	issuer   string
	provider *oidc.Provider
}

func (g *genericTokenVerifier) VerifyIDToken(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
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

	return verifier.Verify(ctx, rawIDToken)
}
