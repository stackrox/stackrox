package complianceoperator

import (
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/k8sapi"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher/availability"
)

type availabilityChecker interface {
	Available(client.Interface) bool
	AppendToCRDWatcher(availability.CrdWatcher) error
	GetResources() []k8sapi.APIResource
}

// NewComplianceOperatorAvailabilityChecker creates a new AvailabilityChecker
func NewComplianceOperatorAvailabilityChecker() availabilityChecker {
	resources := []k8sapi.APIResource{
		complianceoperator.Profile,
		complianceoperator.Rule,
		complianceoperator.ScanSetting,
		complianceoperator.ScanSettingBinding,
		complianceoperator.ComplianceScan,
		complianceoperator.ComplianceSuite,
		complianceoperator.ComplianceCheckResult,
		complianceoperator.TailoredProfile,
		complianceoperator.ComplianceRemediation,
	}
	return availability.NewChecker(complianceoperator.GetGroupVersion(), resources)
}
