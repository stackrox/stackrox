package alertmanager

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// An AlertFilterOption modifies the query builder to filter existing alerts in the DB.
type AlertFilterOption interface {
	apply(*search.QueryBuilder)
	specifiedPolicyID() string
}

type alertFilterOptionImpl struct {
	applyFunc func(*search.QueryBuilder)
	policyID  string
}

func (a *alertFilterOptionImpl) apply(qb *search.QueryBuilder) {
	a.applyFunc(qb)
}

func (a *alertFilterOptionImpl) specifiedPolicyID() string {
	return a.policyID
}

// WithPolicyID returns an AlertFilterOption that filters by policy id.
func WithPolicyID(policyID string) AlertFilterOption {
	return &alertFilterOptionImpl{
		applyFunc: func(qb *search.QueryBuilder) {
			qb.AddExactMatches(search.PolicyID, policyID)
		},
		policyID: policyID,
	}
}

// WithDeploymentIDs returns an AlertFilterOption that filters by deployment id.
func WithDeploymentIDs(deploymentIDs ...string) AlertFilterOption {
	return &alertFilterOptionImpl{
		applyFunc: func(qb *search.QueryBuilder) {
			qb.AddExactMatches(search.DeploymentID, deploymentIDs...)
		},
	}
}

// WithLifecycleStage returns an AlertFilterOptions that filters by lifecycle stage.
func WithLifecycleStage(lifecycleStage storage.LifecycleStage) AlertFilterOption {
	return &alertFilterOptionImpl{
		applyFunc: func(qb *search.QueryBuilder) {
			qb.AddStrings(search.LifecycleStage, lifecycleStage.String())
		},
	}
}
