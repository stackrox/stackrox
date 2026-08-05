package fio

import (
	"github.com/stackrox/rox/pkg/policyreport"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher/availability"
)

// NewAvailabilityChecker creates an availability checker for FileIntegrity CRDs.
func NewAvailabilityChecker() availability.Checker {
	resources := policyreport.GetFIORequiredResources()
	return availability.NewChecker(policyreport.GetFIOGroupVersion(), resources)
}
