package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newRemoteMatcher)
}

func newRemoteMatcher(policy *v1.Policy) (Matcher, error) {
	remotePolicy := policy.GetFields().GetImageName().GetRemote()
	if remotePolicy == "" {
		return nil, nil
	}

	remoteRegex, err := utils.CompileStringRegex(remotePolicy)
	if err != nil {
		return nil, err
	}
	matcher := &remoteMatcherImpl{remoteRegex}
	return matcher.match, nil
}

type remoteMatcherImpl struct {
	remoteRegex *regexp.Regexp
}

func (p *remoteMatcherImpl) match(name *storage.ImageName) []*v1.Alert_Violation {
	remote := name.GetRemote()

	var violations []*v1.Alert_Violation
	if remote != "" && p.remoteRegex.MatchString(remote) {
		v := &v1.Alert_Violation{
			Message: fmt.Sprintf("Image remote matched: %s", p.remoteRegex),
		}
		violations = append(violations, v)
	}
	return violations
}
