package service

import (
	"time"

	clusterStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/jwt"
	roleStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	//#nosec G101 -- This constant is only there as a source identifier, not as a credential.
	internalTokenId = "https://stackrox.io/jwt-sources#internal-rox-tokens"
	//#nosec G101 -- This constant is only there as a source type name, not as a credential.
	internalToken = "internal-token"

	defaultSourcePurgeInterval = 5 * time.Minute
)

var (
	once sync.Once
	s    Service
)

// Singleton returns a new auth service instance.
func Singleton() Service {
	once.Do(func() {
		issuerMgr := newIssuerManager(jwt.IssuerFactorySingleton(), defaultSourcePurgeInterval)
		issuerMgr.Start()
		s = &serviceImpl{
			issuerManager: issuerMgr,
			roleManager: &roleManager{
				clusterStore: clusterStore.Singleton(),
				roleStore:    roleStore.Singleton(),
			},
			now:    time.Now,
			policy: defaultTokenPolicy(),
		}
	})
	return s
}
