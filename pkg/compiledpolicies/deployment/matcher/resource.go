package matcher

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newResourceMatcher)
}

func newResourceMatcher(policy *v1.Policy) (Matcher, error) {
	resourcePolicy := policy.GetFields().GetTotalResourcePolicy()
	if resourcePolicy == nil {
		return nil, nil
	}
	matcher := &resourceMatcherImpl{resourcePolicy: resourcePolicy}
	return matcher.match, nil
}

type resourceMatcherImpl struct {
	resourcePolicy *v1.ResourcePolicy
}

func (p *resourceMatcherImpl) match(deployment *v1.Deployment) []*v1.Alert_Violation {
	var resource v1.Resources
	for _, c := range deployment.GetContainers() {
		resource.CpuCoresRequest += c.GetResources().GetCpuCoresRequest() * float32(deployment.GetReplicas())
		resource.CpuCoresLimit += c.GetResources().GetCpuCoresLimit() * float32(deployment.GetReplicas())
		resource.MemoryMbRequest += c.GetResources().GetMemoryMbRequest() * float32(deployment.GetReplicas())
		resource.MemoryMbLimit += c.GetResources().GetMemoryMbLimit() * float32(deployment.GetReplicas())
	}
	return utils.MatchResources(p.resourcePolicy, &resource, "deployment")
}
