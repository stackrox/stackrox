package resolvers

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("PolicyCounter", []string{
			"total: Int!",
			"low: Int!",
			"medium: Int!",
			"high: Int!",
			"critical: Int!",
		}),
	)
}

// PolicyCounterResolver returns the counts of policies in a couple different buckets.
type PolicyCounterResolver struct {
	total    int32
	low      int32
	medium   int32
	high     int32
	critical int32
}

// Total returns the total number of violated policies.
func (evr *PolicyCounterResolver) Total(_ context.Context) int32 {
	return evr.total
}

// Low returns the total number of low severity violated policies.
func (evr *PolicyCounterResolver) Low(_ context.Context) int32 {
	return evr.low
}

// Medium returns the total number of moderate severity violated policies.
func (evr *PolicyCounterResolver) Medium(_ context.Context) int32 {
	return evr.medium
}

// High returns the total number of important severity violated policies.
func (evr *PolicyCounterResolver) High(_ context.Context) int32 {
	return evr.high
}

// Critical returns the total number of critical severity violated policies.
func (evr *PolicyCounterResolver) Critical(_ context.Context) int32 {
	return evr.critical
}

// Static helpers.
//////////////////

func mapListAlertsToPolicySeverityCount(alerts []*storage.ListAlert) *PolicyCounterResolver {
	counter := &PolicyCounterResolver{}
	policyIDs := set.NewStringSet()
	for _, alert := range alerts {
		if alert.GetState() != storage.ViolationState_ACTIVE {
			continue
		}
		policy := alert.GetPolicy()
		if !policyIDs.Add(policy.GetId()) {
			continue
		}
		incPolicyCounter(counter, policy.GetSeverity())
	}
	return counter
}

func mapListAlertPoliciesToPolicySeverityCount(policies []*storage.ListAlertPolicy) *PolicyCounterResolver {
	counter := &PolicyCounterResolver{}
	policyIDs := set.NewStringSet()
	for _, policy := range policies {
		if !policyIDs.Add(policy.GetId()) {
			continue
		}
		incPolicyCounter(counter, policy.GetSeverity())
	}
	return counter
}

func incPolicyCounter(counter *PolicyCounterResolver, severity storage.Severity) {
	counter.total++
	switch severity {
	case storage.Severity_LOW_SEVERITY:
		counter.low++
	case storage.Severity_MEDIUM_SEVERITY:
		counter.medium++
	case storage.Severity_HIGH_SEVERITY:
		counter.high++
	case storage.Severity_CRITICAL_SEVERITY:
		counter.critical++
	}
}
