package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newRequiredAnnotationMatcher)
}

func newRequiredAnnotationMatcher(policy *storage.Policy) (Matcher, error) {
	env := policy.GetFields().GetRequiredAnnotation()
	if env == nil {
		return nil, nil
	}
	if env.GetKey() == "" && env.GetValue() == "" {
		return nil, fmt.Errorf("both key and value cannot be empty (environment policy)")
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

func (p *requiredAnnotationMatcherImpl) match(deployment *storage.Deployment) []*storage.Alert_Violation {
	return utils.MatchRequiredMap(deployment.GetAnnotations(), p.key, p.value, "annotation")
}
