package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newProcessMatcher)
}

func newProcessMatcher(policy *v1.Policy) (Matcher, error) {
	env := policy.GetFields().GetProcessPolicy()
	if env == nil {
		return nil, nil
	}
	if env.GetName() == "" && env.GetArgs() == "" {
		return nil, fmt.Errorf("both name and args cannot be empty (process policy)")
	}

	name, err := utils.CompileStringRegex(env.GetName())
	if err != nil {
		return nil, err
	}

	args, err := utils.CompileStringRegex(env.GetArgs())
	if err != nil {
		return nil, err
	}
	matcher := &processMatcherImpl{name: name, args: args}
	return matcher.match, nil
}

type processMatcherImpl struct {
	name *regexp.Regexp
	args *regexp.Regexp
}

func (m *processMatcherImpl) match(deployment *v1.Deployment) (violations []*v1.Alert_Violation) {
	violations = make([]*v1.Alert_Violation, 0)
	/*
		IMPORTANT: deployment.GetProcesses() needs to be manually populated before match is called.
	*/
	for _, p := range deployment.GetProcesses() {
		n := p.GetSignal().GetName()
		a := p.GetSignal().GetArgs()
		if m.name != nil && m.args != nil {
			if m.name.MatchString(n) && m.args.MatchString(a) {
				violations = append(violations, generateProcessViolation(n, a))
			}
		} else if m.name != nil {
			if m.name.MatchString(n) {
				violations = append(violations, generateProcessViolation(n, a))
			}
		} else if m.args != nil {
			if m.args.MatchString(n) {
				violations = append(violations, generateProcessViolation(n, a))
			}
		}
	}
	return
}

func generateProcessViolation(name, args string) *v1.Alert_Violation {
	var argsMessage string
	if args != "" {
		argsMessage = fmt.Sprintf(" with arguments '%s'", args)
	}
	return &v1.Alert_Violation{
		Message: fmt.Sprintf("Detected running process '%s'%s", name, argsMessage),
	}
}
