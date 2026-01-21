package service

import (
	"time"

	"github.com/stackrox/rox/central/jwt"
	"github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/pkg/auth/authproviders/tokenbased"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	//#nosec G101 -- This constant is only there as a source identifier, not as a credential.
	internalTokenId = "https://stackrox.io/jwt-sources#internal-rox-tokens"
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
		s = &serviceImpl{issuer: getTokenIssuer(), roleManager: &roleManager{roleStore: datastore.Singleton()}, now: time.Now}
	})
	return s
}

func getTokenSource() tokens.Source {
	return tokenbased.NewTokenAuthProvider(
		internalTokenId,
		internalToken,
		internalToken,
		tokenbased.WithRevocationLayer(tokens.NewRevocationLayer()),
	)
}

func getTokenIssuer() tokens.Issuer {
	issuer, err := jwt.IssuerFactorySingleton().CreateIssuer(getTokenSource())
	utils.Must(err)
	return issuer
}
