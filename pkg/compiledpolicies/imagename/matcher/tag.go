package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, NewTagMatcher)
}

// NewTagMatcher should not be exposed.
func NewTagMatcher(policy *v1.Policy) (Matcher, error) {
	tagPolicy := policy.GetFields().GetImageName().GetTag()
	if tagPolicy == "" {
		return nil, nil
	}

	tagRegex, err := utils.CompileStringRegex(tagPolicy)
	if err != nil {
		return nil, err
	}
	matcher := &tagMatcherImpl{tagRegex}
	return matcher.match, nil
}

type tagMatcherImpl struct {
	tagRegex *regexp.Regexp
}

func (p *tagMatcherImpl) match(name *storage.ImageName) []*v1.Alert_Violation {
	var violations []*v1.Alert_Violation
	if name.GetTag() != "" && p.tagRegex.MatchString(name.GetTag()) {
		v := &v1.Alert_Violation{
			Message: fmt.Sprintf("Image tag matched: %s", p.tagRegex),
		}
		violations = append(violations, v)
	}
	return violations
}
