package cache

import (
	"github.com/stackrox/rox/pkg/sync"
)

var (
	activeOnce        sync.Once
	activeReqInstance VulnReqCache

	pendingOnce        sync.Once
	pendingReqInstance VulnReqCache
)

// ActiveReqsCacheSingleton provides the instance of VulnReqCache that holds active vuln reqs.
func ActiveReqsCacheSingleton() VulnReqCache {
	activeOnce.Do(func() {
		activeReqInstance = New()
	})
	return activeReqInstance
}

// PendingReqsCacheSingleton provides the instance of VulnReqCache that holds pending vuln reqs.
func PendingReqsCacheSingleton() VulnReqCache {
	pendingOnce.Do(func() {
		pendingReqInstance = New()
	})
	return pendingReqInstance
}

// New returns an initialized vulnerability requests cache.
func New() VulnReqCache {
	return &vulnReqCacheImpl{
		vulnReqByScope:  make(map[string]map[string]*slimRequest),
		scopeByVulnReqs: make(map[string]string),
	}
}
