package virtualmachine

import (
	"github.com/stackrox/rox/pkg/virtualmachine"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher/availability"
)

// NewAvailabilityChecker creates a new AvailabilityChecker
func NewAvailabilityChecker() availability.Checker {
	resources := virtualmachine.GetRequiredResources()
	return availability.NewChecker(virtualmachine.GetGroupVersion(), resources)
}
