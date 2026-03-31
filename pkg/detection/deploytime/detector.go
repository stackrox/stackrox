package deploytime

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/detection"
)

// DetectOption configures the behavior of Detect.
type DetectOption func(*detectionConfig)

type detectionConfig struct {
	enforcementOnly bool
	policyFilters   []detection.FilterOption
}

// WithEnforcementOnly skips alerts for policies that have no enforcement action.
func WithEnforcementOnly() DetectOption {
	return func(cfg *detectionConfig) {
		cfg.enforcementOnly = true
	}
}

// WithPolicyFilters adds policy filters that each policy must pass before evaluation.
func WithPolicyFilters(filters ...detection.FilterOption) DetectOption {
	return func(cfg *detectionConfig) {
		cfg.policyFilters = append(cfg.policyFilters, filters...)
	}
}

// Detector provides an interface for getting and managing alerts and enforcements on deployments.
type Detector interface {
	PolicySet() detection.PolicySet

	Detect(ctx context.Context, enhancedDeployment booleanpolicy.EnhancedDeployment, opts ...DetectOption) ([]*storage.Alert, error)
}

// NewDetector returns a new instance of a Detector.
func NewDetector(policySet detection.PolicySet) Detector {
	return &detectorImpl{
		policySet: policySet,
	}
}
