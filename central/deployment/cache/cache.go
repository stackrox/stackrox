package cache

import (
	"time"

	"github.com/stackrox/stackrox/pkg/expiringcache"
)

const (
	deletedDeploymentsRetentionPeriod = 2 * time.Minute
)

var (
	cache = expiringcache.NewExpiringCache(deletedDeploymentsRetentionPeriod)
)

// DeletedDeploymentCacheSingleton returns a global expiringcache for deployments that have been recently deleted
func DeletedDeploymentCacheSingleton() expiringcache.Cache {
	return cache
}
