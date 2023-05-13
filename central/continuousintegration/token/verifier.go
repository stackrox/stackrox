package token

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
)

type gitHubVerifier struct {
	verifier *oidc.IDTokenVerifier
}

func newGitHubVerifier() *gitHubVerifier {
	provider, err := oidc.NewProvider(context.Background(), "https://token.actions.githubusercontent.com")
	if err != nil {
		panic(err)
	}

	idTokenVerifier := provider.Verifier(&oidc.Config{
		SupportedSigningAlgs: nil,
		// The audience is non-deterministic as it will be the repository owner's URL (which may be a magnitude of URLs)
		// and can be customized, hence skipping this.
		// See https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect#understanding-the-oidc-token
		SkipClientIDCheck: true,
	})

	return &gitHubVerifier{
		verifier: idTokenVerifier,
	}
}

func (g *gitHubVerifier) VerifyToken(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	idToken, err := g.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}
	return idToken, nil
}
