package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newEnvironmentMatcher)
}

func newEnvironmentMatcher(policy *v1.Policy) (Matcher, error) {
	env := policy.GetFields().GetEnv()
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

	matcher := &environmentMatcherImpl{key: key, value: value}
	return matcher.match, nil
}

type environmentMatcherImpl struct {
	key   *regexp.Regexp
	value *regexp.Regexp
}

func (p *environmentMatcherImpl) match(container *storage.Container) []*v1.Alert_Violation {
	config := container.GetConfig()
	var violations []*v1.Alert_Violation
	for _, env := range config.GetEnv() {
		if p.key != nil && p.value != nil {
			if p.key.MatchString(env.GetKey()) && p.value.MatchString(env.GetValue()) {
				violations = append(violations, &v1.Alert_Violation{
					Message: fmt.Sprintf("Container Environment (key='%s', value='%s') matched configured policy (key='%s', value='%s')", env.GetKey(), env.GetValue(), p.key, p.value),
				})
			}
		} else if p.key != nil {
			if p.key.MatchString(env.GetKey()) {
				violations = append(violations, &v1.Alert_Violation{
					Message: fmt.Sprintf("Container Environment (key='%s', value='%s') matched configured policy (key='%s')", env.GetKey(), env.GetValue(), p.key),
				})
			}
		} else if p.value != nil {
			if p.value.MatchString(env.GetValue()) {
				violations = append(violations, &v1.Alert_Violation{
					Message: fmt.Sprintf("Container Environment (key='%s', value='%s') matched configured policy (value='%s')", env.GetKey(), env.GetValue(), p.value),
				})
			}
		}
	}
	return violations
}
