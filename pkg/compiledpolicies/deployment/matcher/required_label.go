package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newRequiredLabelMatcher)
}

func newRequiredLabelMatcher(policy *v1.Policy) (Matcher, error) {
	env := policy.GetFields().GetRequiredLabel()
	if env == nil {
		return nil, nil
	}
	if env.GetKey() == "" && env.GetValue() == "" {
		return nil, fmt.Errorf("Both key and value cannot be empty (environment policy)")
	}

	key, err := utils.CompileStringRegex(env.GetKey())
	if err != nil {
		return nil, err
	}

	value, err := utils.CompileStringRegex(env.GetValue())
	if err != nil {
		return nil, err
	}
	matcher := &requiredLabelMatcherMatcherImpl{key: key, value: value}
	return matcher.match, nil
}

type requiredLabelMatcherMatcherImpl struct {
	key   *regexp.Regexp
	value *regexp.Regexp
}

func (p *requiredLabelMatcherMatcherImpl) match(deployment *v1.Deployment) []*v1.Alert_Violation {
	return utils.MatchRequiredMap(deployment.GetLabels(), p.key, p.value, "label")
}
