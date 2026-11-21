package policyreport

import (
	"github.com/stackrox/rox/pkg/policyreport"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher/availability"
)

// NewAvailabilityChecker creates an availability checker for PolicyReport CRDs.
func NewAvailabilityChecker() availability.Checker {
	resources := policyreport.GetRequiredResources()
	return availability.NewChecker(policyreport.GetGroupVersion(), resources)
}
