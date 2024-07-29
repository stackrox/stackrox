package cache

import (
	"time"

	"github.com/stackrox/rox/pkg/expiringcache"
)

const (
	deletedDeploymentsRetentionPeriod = 2 * time.Minute
)

var (
	cache = &deploymentCache{
		cache: expiringcache.NewExpiringCache[string, struct{}](deletedDeploymentsRetentionPeriod),
	}
)

type DeletedDeployments interface {
	Add(id string)
	Contains(id string) bool
}

type deploymentCache struct {
	cache expiringcache.Cache[string, struct{}]
}

func (c *deploymentCache) Add(id string) {
	cache.cache.Add(id, struct{}{})
}

func (c *deploymentCache) Contains(id string) bool {
	_, ok := cache.cache.Get(id)
	return ok
}

// DeletedDeploymentsSingleton returns a global expiringcache for deployments that have been recently deleted.
func DeletedDeploymentsSingleton() DeletedDeployments {
	return cache
}
