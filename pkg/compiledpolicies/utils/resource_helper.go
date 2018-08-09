package utils

import (
	"fmt"
	"math"

	"github.com/stackrox/rox/generated/api/v1"
)

// MatchResources returns violations based on the input resource policy and actual resource configuration.
func MatchResources(policy *v1.ResourcePolicy, resource *v1.Resources, identifier string) []*v1.Alert_Violation {
	matchFunctions := []func(*v1.ResourcePolicy, *v1.Resources, string) ([]*v1.Alert_Violation, bool){
		matchCPUResourceRequest,
		matchCPUResourceLimit,
		matchMemoryResourceRequest,
		matchMemoryResourceLimit,
	}

	// OR the violations together
	var violations []*v1.Alert_Violation
	for _, f := range matchFunctions {
		vs, _ := f(policy, resource, identifier)
		violations = append(violations, vs...)
	}
	return violations
}

func matchCPUResourceRequest(rp *v1.ResourcePolicy, resources *v1.Resources, id string) (violations []*v1.Alert_Violation, policyExists bool) {
	violations, policyExists = matchNumericalPolicy("CPU resource request",
		id, resources.GetCpuCoresRequest(), rp.GetCpuResourceRequest())
	return
}

func matchCPUResourceLimit(rp *v1.ResourcePolicy, resources *v1.Resources, id string) (violations []*v1.Alert_Violation, policyExists bool) {
	violations, policyExists = matchNumericalPolicy("CPU resource limit",
		id, resources.GetCpuCoresLimit(), rp.GetCpuResourceLimit())
	return
}

func matchMemoryResourceRequest(rp *v1.ResourcePolicy, resources *v1.Resources, id string) (violations []*v1.Alert_Violation, policyExists bool) {
	violations, policyExists = matchNumericalPolicy("Memory resource request",
		id, resources.GetMemoryMbRequest(), rp.GetMemoryResourceRequest())
	return
}

func matchMemoryResourceLimit(rp *v1.ResourcePolicy, resources *v1.Resources, id string) (violations []*v1.Alert_Violation, policyExists bool) {
	violations, policyExists = matchNumericalPolicy("Memory resource limit",
		id, resources.GetMemoryMbLimit(), rp.GetMemoryResourceLimit())
	return
}

func matchNumericalPolicy(prefix, id string, value float32, p *v1.ResourcePolicy_NumericalPolicy) (violations []*v1.Alert_Violation, policyExists bool) {
	if p == nil {
		return
	}
	policyExists = true
	var comparatorFunc func(x, y float32) bool
	var comparatorString string
	switch p.GetOp() {
	case v1.Comparator_LESS_THAN:
		comparatorFunc = func(x, y float32) bool { return x < y }
		comparatorString = "less than"
	case v1.Comparator_LESS_THAN_OR_EQUALS:
		comparatorFunc = func(x, y float32) bool { return x <= y }
		comparatorString = "less than or equal to"
	case v1.Comparator_EQUALS:
		comparatorFunc = func(x, y float32) bool { return math.Abs(float64(x-y)) <= 1e-5 }
		comparatorString = "equal to"
	case v1.Comparator_GREATER_THAN_OR_EQUALS:
		comparatorFunc = func(x, y float32) bool { return x >= y }
		comparatorString = "greater than or equal to"
	case v1.Comparator_GREATER_THAN:
		comparatorFunc = func(x, y float32) bool { return x > y }
		comparatorString = "greater than"
	}
	if comparatorFunc(value, p.GetValue()) {
		violations = append(violations, &v1.Alert_Violation{
			Message: fmt.Sprintf("The %s of %0.2f for %s is %s the threshold of %v", prefix, value,
				id, comparatorString, p.GetValue()),
		})
	}
	return
}
