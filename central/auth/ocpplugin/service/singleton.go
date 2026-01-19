package service

import (
	"time"

	"github.com/stackrox/rox/central/jwt"
	"github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/pkg/auth/authproviders/tokenbasedsource"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	id = "https://stackrox.io/jwt-sources#ocp-rox-tokens"
	//#nosec G101 -- This constant is only there as a source type name, not as a credential.
	ocpToken = "ocp-plugin-token"
)

var (
	once sync.Once
	s    Service
)

// Singleton returns a new auth service instance.
func Singleton() Service {
	once.Do(func() {
		s = &serviceImpl{issuer: getTokenIssuer(), roleStore: datastore.Singleton(), now: time.Now}
	})
	return s
}

func getTokenSource() tokens.Source {
	return tokenbasedsource.NewTokenSource(
		id,
		ocpToken,
		ocpToken,
		tokenbasedsource.WithRevocationLayer(tokens.NewRevocationLayer()),
	)
}

func getTokenIssuer() tokens.Issuer {
	issuer, err := jwt.IssuerFactorySingleton().CreateIssuer(getTokenSource())
	utils.Must(err)
	return issuer
}
