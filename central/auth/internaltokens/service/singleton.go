package service

import (
	"time"

	clusterStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/jwt"
	roleStore "github.com/stackrox/rox/central/role/datastore"
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
		s = &serviceImpl{
			issuer: getTokenIssuer(jwt.IssuerFactorySingleton()),
			roleManager: &roleManager{
				clusterStore: clusterStore.Singleton(),
				roleStore:    roleStore.Singleton(),
			},
			now: time.Now,
		}
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

func getTokenIssuer(issuerFactory tokens.IssuerFactory) tokens.Issuer {
	issuer, err := issuerFactory.CreateIssuer(getTokenSource())
	utils.Must(err)
	return issuer
}
