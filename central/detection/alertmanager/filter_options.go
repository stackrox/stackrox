package alertmanager

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// An AlertFilterOption modifies the query builder to filter existing alerts in the DB.
// Clients should use the helper functions below to create one instead of creating them on their own.
type AlertFilterOption func(*search.QueryBuilder)

// WithPolicyID returns an AlertFilterOption that filters by policy id.
func WithPolicyID(policyID string) AlertFilterOption {
	return func(qb *search.QueryBuilder) {
		qb.AddExactMatches(search.PolicyID, policyID)
	}
}

// WithDeploymentIDs returns an AlertFilterOption that filters by deployment id.
func WithDeploymentIDs(deploymentIDs ...string) AlertFilterOption {
	return func(qb *search.QueryBuilder) {
		qb.AddExactMatches(search.DeploymentID, deploymentIDs...)
	}
}

// WithLifecycleStage returns an AlertFilterOptions that filters by lifecycle stage.
func WithLifecycleStage(lifecycleStage v1.LifecycleStage) AlertFilterOption {
	return func(qb *search.QueryBuilder) {
		qb.AddStrings(search.LifecycleStage, lifecycleStage.String())
	}
}
