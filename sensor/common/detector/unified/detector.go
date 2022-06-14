package unified

import (
	"github.com/stackrox/stackrox/generated/internalapi/sensor"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/booleanpolicy"
	"github.com/stackrox/stackrox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/stackrox/pkg/detection"
	"github.com/stackrox/stackrox/pkg/detection/deploytime"
	"github.com/stackrox/stackrox/pkg/detection/runtime"
	"github.com/stackrox/stackrox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Detector is a thin layer atop the other detectors that provides a unified interface.
type Detector interface {
	ReconcilePolicies(newList []*storage.Policy)
	DetectDeployment(ctx deploytime.DetectionContext, deployment booleanpolicy.EnhancedDeployment) []*storage.Alert
	DetectProcess(enhancedDeployment booleanpolicy.EnhancedDeployment, processIndicator *storage.ProcessIndicator, processNotInBaseline bool) []*storage.Alert
	DetectKubeEventForDeployment(enhancedDeployment booleanpolicy.EnhancedDeployment, kubeEvent *storage.KubernetesEvent) []*storage.Alert
	DetectNetworkFlowForDeployment(enhancedDeployment booleanpolicy.EnhancedDeployment, flow *augmentedobjs.NetworkFlowDetails) []*storage.Alert
	DetectAuditLogEvents(auditEvent *sensor.AuditEvents) []*storage.Alert
}

// NewDetector returns a new detector.
func NewDetector() Detector {
	return &detectorImpl{
		deploytimeDetector: deploytime.NewDetector(detection.NewPolicySet()),
		runtimeDetector:    runtime.NewDetector(detection.NewPolicySet()),
	}
}
