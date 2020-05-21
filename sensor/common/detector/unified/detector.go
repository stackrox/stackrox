package unified

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/detection/runtime"
	"github.com/stackrox/rox/pkg/features"
	options "github.com/stackrox/rox/pkg/search/options/deployments"
	"github.com/stackrox/rox/pkg/searchbasedpolicies/matcher"
)

// Detector is a thin layer atop the other detectors that provides a unified interface.
type Detector interface {
	ReconcilePolicies(newList []*storage.Policy)
	DetectDeployment(ctx deploytime.DetectionContext, deployment *storage.Deployment, images []*storage.Image) []*storage.Alert
	DetectProcess(deployment *storage.Deployment, images []*storage.Image, processIndicator *storage.ProcessIndicator, processOutsideWhitelist bool) []*storage.Alert
}

// NewDetector returns a new detector.
func NewDetector() Detector {
	if !features.BooleanPolicyLogic.Enabled() {
		return newLegacyDetector()
	}
	return &detectorImpl{
		deploytimeDetector: deploytime.NewDetector(detection.NewPolicySet(detection.NewPolicyCompiler())),
		runtimeDetector:    runtime.NewDetector(detection.NewPolicySet(detection.NewPolicyCompiler())),
	}
}

func newLegacyDetector() Detector {
	builder := matcher.NewBuilder(
		matcher.NewRegistry(
			nil,
		),
		options.OptionsMap,
	)
	return &legacyDetectorImpl{
		deploytimeDetector:       deploytime.NewDetector(detection.NewPolicySet(detection.NewLegacyPolicyCompiler(builder))),
		runtimeDetector:          runtime.NewDetector(detection.NewPolicySet(detection.NewLegacyPolicyCompiler(builder))),
		runtimeWhitelistDetector: runtime.NewDetector(detection.NewPolicySet(detection.NewLegacyPolicyCompiler(builder))),
	}
}
