package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newDestinationMatcher)
}

func newDestinationMatcher(policy *storage.Policy) (Matcher, error) {
	destination := policy.GetFields().GetVolumePolicy().GetDestination()
	if destination == "" {
		return nil, nil
	}

	destinationRegex, err := utils.CompileStringRegex(destination)
	if err != nil {
		return nil, err
	}
	matcher := &destinationMatcherImpl{destinationRegex}
	return matcher.match, nil
}

type destinationMatcherImpl struct {
	destinationRegex *regexp.Regexp
}

func (p *destinationMatcherImpl) match(volume *storage.Volume) []*storage.Alert_Violation {
	var violations []*storage.Alert_Violation
	if p.destinationRegex.MatchString(volume.GetDestination()) {
		v := &storage.Alert_Violation{
			Message: fmt.Sprintf("Volume destination matched: %s", p.destinationRegex),
		}
		violations = append(violations, v)
	}
	return violations
}
