package watcher

import "github.com/stackrox/rox/pkg/set"

// Status represents the state of a watcher. Available means all the Resources are available in the cluster.
type Status struct {
	// Available is 'true' is all the Resources are available
	Available bool
	// Resources a StringSet with all the resources that are being watched
	Resources set.FrozenStringSet
}
