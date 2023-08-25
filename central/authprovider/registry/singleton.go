package registry

import (
	authProviderDS "github.com/stackrox/rox/central/authprovider/datastore"
	"github.com/stackrox/rox/central/jwt"
	"github.com/stackrox/rox/central/role/mapper"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	ssoURLPathPrefix = "/sso/"
	//#nosec G101 -- This is a false positive
	tokenRedirectURLPath = "/auth/response/generic"
)

var (
	once     sync.Once
	registry authproviders.Registry
)

func initialize() {
	registry = authproviders.NewStoreBackedRegistry(
		ssoURLPathPrefix, tokenRedirectURLPath,
		authProviderDS.Singleton(), jwt.IssuerFactorySingleton(),
		mapper.FactorySingleton())
}

// Singleton returns the auth providers registry.
func Singleton() authproviders.Registry {
	once.Do(initialize)
	return registry
}
