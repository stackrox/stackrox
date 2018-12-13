package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newSourceMatcher)
}

func newSourceMatcher(policy *storage.Policy) (Matcher, error) {
	source := policy.GetFields().GetVolumePolicy().GetSource()
	if source == "" {
		return nil, nil
	}

	sourceRegex, err := utils.CompileStringRegex(source)
	if err != nil {
		return nil, err
	}
	matcher := &sourceMatcherImpl{sourceRegex}
	return matcher.match, nil
}

type sourceMatcherImpl struct {
	sourceRegex *regexp.Regexp
}

func (p *sourceMatcherImpl) match(volume *storage.Volume) []*storage.Alert_Violation {
	var violations []*storage.Alert_Violation
	if p.sourceRegex.MatchString(volume.GetSource()) {
		v := &storage.Alert_Violation{
			Message: fmt.Sprintf("Volume source matched: %s", p.sourceRegex),
		}
		violations = append(violations, v)
	}
	return violations
}
