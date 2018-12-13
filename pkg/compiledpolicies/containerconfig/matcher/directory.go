package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newDirectoryMatcher)
}

func newDirectoryMatcher(policy *storage.Policy) (Matcher, error) {
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

func (p *directoryMatcherImpl) match(config *storage.ContainerConfig) []*storage.Alert_Violation {
	var violations []*storage.Alert_Violation
	if p.directoryRegex.MatchString(config.GetDirectory()) {
		v := &storage.Alert_Violation{
			Message: fmt.Sprintf("Directory matched configs policy: %s", p.directoryRegex),
		}
		violations = append(violations, v)
	}
	return violations
}
