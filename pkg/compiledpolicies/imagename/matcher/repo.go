package matcher

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newRepoMatcher)
}

func newRepoMatcher(policy *v1.Policy) (Matcher, error) {
	repoPolicy := policy.GetFields().GetImageName().GetRepo()
	if repoPolicy == "" {
		return nil, nil
	}

	repoRegex, err := utils.CompileStringRegex(repoPolicy)
	if err != nil {
		return nil, err
	}
	matcher := &repoMatcherImpl{repoRegex}
	return matcher.match, nil
}

type repoMatcherImpl struct {
	repoRegex *regexp.Regexp
}

func (p *repoMatcherImpl) match(name *v1.ImageName) []*v1.Alert_Violation {
	var repo string
	remoteSplit := strings.Split(name.GetRemote(), "/")
	if len(remoteSplit) < 2 {
		repo = remoteSplit[0]
	} else {
		repo = remoteSplit[1]
	}

	var violations []*v1.Alert_Violation
	if repo != "" && p.repoRegex.MatchString(repo) {
		v := &v1.Alert_Violation{
			Message: fmt.Sprintf("Image repo matched: %s", p.repoRegex),
		}
		violations = append(violations, v)
	}
	return violations
}
