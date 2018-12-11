package matcher

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newResourceMatcher)
}

func newResourceMatcher(policy *v1.Policy) (Matcher, error) {
	resourcePolicy := policy.GetFields().GetContainerResourcePolicy()
	if resourcePolicy == nil {
		return nil, nil
	}

	matcher := &resourceMatcherImpl{resourcePolicy: resourcePolicy}
	return matcher.match, nil
}

type resourceMatcherImpl struct {
	resourcePolicy *v1.ResourcePolicy
}

func (p *resourceMatcherImpl) match(container *storage.Container) []*v1.Alert_Violation {
	return utils.MatchResources(p.resourcePolicy, container.GetResources(), fmt.Sprintf("container %s", container.GetImage().GetName().GetRemote()))
}
