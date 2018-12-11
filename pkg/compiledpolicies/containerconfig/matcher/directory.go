package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newDirectoryMatcher)
}

func newDirectoryMatcher(policy *v1.Policy) (Matcher, error) {
	directory := policy.GetFields().GetDirectory()
	if directory == "" {
		return nil, nil
	}

	directoryRegex, err := utils.CompileStringRegex(directory)
	if err != nil {
		return nil, err
	}
	matcher := &directoryMatcherImpl{directoryRegex}
	return matcher.match, nil
}

type directoryMatcherImpl struct {
	directoryRegex *regexp.Regexp
}

func (p *directoryMatcherImpl) match(config *storage.ContainerConfig) []*v1.Alert_Violation {
	var violations []*v1.Alert_Violation
	if p.directoryRegex.MatchString(config.GetDirectory()) {
		v := &v1.Alert_Violation{
			Message: fmt.Sprintf("Directory matched configs policy: %s", p.directoryRegex),
		}
		violations = append(violations, v)
	}
	return violations
}
