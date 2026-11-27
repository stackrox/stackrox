package service

import (
	"time"

	"github.com/stackrox/rox/central/jwt"
	roleDS "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/pkg/auth/authproviders/tokenbased"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	//#nosec G101 -- This constant is only there as a source identifier, not as a credential.
	internalTokenId = "https://stackrox.io/jwt-sources#internal-rox-tokens"
	//#nosec G101 -- This constant identifies a token source, not a credential.
	internalHTTPScopedTokenID = "https://stackrox.io/jwt-sources#internal-rox-tokens-http-scoped"
	//#nosec G101 -- This constant is only there as a source type name, not as a credential.
	internalToken = "internal-token"
)

var (
	once sync.Once
	s    Service
)

// Singleton returns a new auth service instance.
func Singleton() Service {
	once.Do(func() {
		issuer, scopedIssuer := getTokenIssuers(jwt.IssuerFactorySingleton())
		s = &serviceImpl{
			issuer:       issuer,
			scopedIssuer: scopedIssuer,
			roleManager:  &roleManager{roleStore: roleDS.Singleton()},
			now:          time.Now,
			policy:       defaultTokenPolicy(),
		}
	})
	return s
}

func getTokenIssuers(factory tokens.IssuerFactory) (tokens.Issuer, tokens.Issuer) {
	revocation := tokens.NewRevocationLayer()
	source := tokenbased.NewTokenAuthProvider(internalTokenId, internalToken, internalToken,
		tokenbased.WithRevocationLayer(revocation))
	scopedSource := tokenbased.NewTokenAuthProvider(internalHTTPScopedTokenID, internalToken, internalToken,
		tokenbased.WithRevocationLayer(revocation),
		tokenbased.WithValidationLayer(tokenbased.NewHTTPScopedValidationLayer(internalHTTPScopedTokenID)))
	issuer, err := factory.CreateIssuer(source)
	utils.Must(err)
	scopedIssuer, err := factory.CreateIssuer(scopedSource)
	utils.Must(err)
	return issuer, scopedIssuer
}

func getTokenSource() tokens.Source {
	return tokenbased.NewTokenAuthProvider(
		internalTokenId,
		internalToken,
		internalToken,
		tokenbased.WithRevocationLayer(tokens.NewRevocationLayer()),
	)
}

func getTokenIssuer(issuerFactory tokens.IssuerFactory) tokens.Issuer {
	issuer, err := issuerFactory.CreateIssuer(getTokenSource())
	utils.Must(err)
	return issuer
}
