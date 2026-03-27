package unified

import (
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/detection/runtime"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Detector is a thin layer atop the other detectors that provides a unified interface.
type Detector interface {
	ReconcilePolicies(newList []*storage.Policy)
	DetectDeployment(deployment booleanpolicy.EnhancedDeployment) []*storage.Alert
	DetectProcess(enhancedDeployment booleanpolicy.EnhancedDeployment, processIndicator *storage.ProcessIndicator, processNotInBaseline bool) []*storage.Alert
	DetectKubeEventForDeployment(enhancedDeployment booleanpolicy.EnhancedDeployment, kubeEvent *storage.KubernetesEvent) []*storage.Alert
	DetectNetworkFlowForDeployment(enhancedDeployment booleanpolicy.EnhancedDeployment, flow *augmentedobjs.NetworkFlowDetails) []*storage.Alert
	DetectAuditLogEvents(auditEvent *sensor.AuditEvents) []*storage.Alert
	DetectNodeFileAccess(node *storage.Node, access *storage.FileAccess) []*storage.Alert
	DetectFileAccessForDeployment(enhancedDeployment booleanpolicy.EnhancedDeployment, fileAccess *storage.FileAccess) []*storage.Alert
}

// NewDetector returns a new detector.
func NewDetector() Detector {
	// TODO: Wire cluster and namespace label providers from Sensor's in-memory stores.
	// For now, passing nil providers means policies with cluster_label/namespace_label scopes will
	// fail closed (not match) in Sensor policy evaluation.
	return &detectorImpl{
		deploytimeDetector: deploytime.NewDetector(detection.NewPolicySet(nil, nil)),
		runtimeDetector:    runtime.NewDetector(detection.NewPolicySet(nil, nil)),
	}
}
