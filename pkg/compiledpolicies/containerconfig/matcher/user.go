package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newUserMatcher)
}

func newUserMatcher(policy *storage.Policy) (Matcher, error) {
	user := policy.GetFields().GetUser()

	if user == "" {
		return nil, nil
	}

	userRegex, err := utils.CompileStringRegex(user)
	if err != nil {
		return nil, err
	}
	matcher := &userMatcherImpl{userRegex}
	return matcher.match, nil
}

type userMatcherImpl struct {
	userRegex *regexp.Regexp
}

func (p *userMatcherImpl) match(config *storage.ContainerConfig) []*storage.Alert_Violation {
	var violations []*storage.Alert_Violation
	if p.userRegex.MatchString(config.GetUser()) {
		v := &storage.Alert_Violation{
			Message: fmt.Sprintf("User matched configs policy: %s", p.userRegex),
		}
		violations = append(violations, v)
	}
	return violations
}
