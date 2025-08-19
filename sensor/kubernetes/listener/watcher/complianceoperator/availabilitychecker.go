package complianceoperator

import (
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/k8sapi"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher/availability"
)

type checker interface {
	Available(client.Interface) bool
	AppendToCRDWatcher(availability.CrdWatcher) error
	GetResources() []k8sapi.APIResource
}

// NewComplianceOperatorAvailabilityChecker creates a new AvailabilityChecker
func NewComplianceOperatorAvailabilityChecker() checker {
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
	return availability.New(complianceoperator.GetGroupVersion(), resources)
}
