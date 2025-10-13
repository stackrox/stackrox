package cache

import (
	"time"

	"github.com/stackrox/rox/pkg/expiringcache"
)

const (
	// 30 seconds for collector to send a message to sensor
	// 30 seconds for sensor to send a message to central
	// 5 minutes for afterglow
	// 1 minute for a safety margin
	deletedDeploymentsRetentionPeriod = 7 * time.Minute
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
