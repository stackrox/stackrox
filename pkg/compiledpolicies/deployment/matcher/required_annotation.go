package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newRequiredAnnotationMatcher)
}

func newRequiredAnnotationMatcher(policy *v1.Policy) (Matcher, error) {
	env := policy.GetFields().GetRequiredAnnotation()
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
	matcher := &requiredAnnotationMatcherImpl{key: key, value: value}
	return matcher.match, nil
}

type requiredAnnotationMatcherImpl struct {
	key   *regexp.Regexp
	value *regexp.Regexp
}

func (p *requiredAnnotationMatcherImpl) match(deployment *v1.Deployment) []*v1.Alert_Violation {
	return utils.MatchRequiredKeyValue(deployment.GetAnnotations(), p.key, p.value, "label")
}
