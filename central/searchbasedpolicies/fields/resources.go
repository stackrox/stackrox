package fields

import (
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var resourcePolicy = builders.NewDisjunctionQueryBuilder(
	builders.NewResourcePolicyBuilder(func(fields *storage.PolicyFields) *storage.NumericalPolicy {
		return fields.GetContainerResourcePolicy().GetCpuResourceLimit()
	}, search.CPUCoresLimit, "CPU resource limit"),
	builders.NewResourcePolicyBuilder(func(fields *storage.PolicyFields) *storage.NumericalPolicy {
		return fields.GetContainerResourcePolicy().GetCpuResourceRequest()
	}, search.CPUCoresRequest, "CPU resource request"),
	builders.NewResourcePolicyBuilder(func(fields *storage.PolicyFields) *storage.NumericalPolicy {
		return fields.GetContainerResourcePolicy().GetMemoryResourceLimit()
	}, search.MemoryLimit, "memory resource limit"),
	builders.NewResourcePolicyBuilder(func(fields *storage.PolicyFields) *storage.NumericalPolicy {
		return fields.GetContainerResourcePolicy().GetMemoryResourceRequest()
	}, search.MemoryRequest, "memory resource request"),
)
