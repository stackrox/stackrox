package watcher

import (
	"fmt"

	"github.com/stackrox/rox/pkg/set"
)

// Status represents the state of a watcher. Available means all the Resources are available in the cluster.
type Status struct {
	// Available is 'true' is all the Resources are available
	Available bool
	// Resources a StringSet with all the resources that are being watched
	Resources set.FrozenStringSet
}

// String returns the Status in a string
func (s *Status) String() string {
	availabilityStr := "available"
	if !s.Available {
		availabilityStr = "unavailable"
	}
	return fmt.Sprintf("Resources [%s] status changed to %s", s.Resources.ElementsString(", "), availabilityStr)
}
