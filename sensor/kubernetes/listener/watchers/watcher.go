package watchers

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watchers/complianceoperator"
)

// Watcher is responsible to watch for a specific resource to be available in the cluster.
// For example: If a CRD is installed after sensor started, the Watcher for that CRD will detect it.
type Watcher interface {
	Watch(*concurrency.Signal, func(string))
}

// WatcherRegistry provides the resource watchers to use.
type WatcherRegistry interface {
	ForComplianceOperatorRules() Watcher
}

// NewWatcherRegistry creates a new WatcherRegistry.
func NewWatcherRegistry(cli client.Interface) WatcherRegistry {
	return &watcherRegistry{
		complianceOperatorRulesWatcher: complianceoperator.NewComplianceOperatorRulesWatcher(cli),
	}
}

type watcherRegistry struct {
	complianceOperatorRulesWatcher *complianceoperator.RulesWatcher
}

// ForComplianceOperatorRules returns the Watcher for the Compliance Operator Rules.
func (r *watcherRegistry) ForComplianceOperatorRules() Watcher {
	return r.complianceOperatorRulesWatcher
}
