package complianceoperator

import (
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher/availability"
)

// NewComplianceOperatorAvailabilityChecker creates a new AvailabilityChecker
func NewComplianceOperatorAvailabilityChecker() availability.Checker {
	resources := complianceoperator.GetRequiredResources()
	return availability.NewChecker(complianceoperator.GetGroupVersion(), resources)
}
