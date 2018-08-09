package matcher

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newNameSpaceMatcher)
}

func newNameSpaceMatcher(policy *v1.Policy) (Matcher, error) {
	namespacePolicy := policy.GetFields().GetImageName().GetNamespace()
	if namespacePolicy == "" {
		return nil, nil
	}

	namespaceRegex, err := utils.CompileStringRegex(namespacePolicy)
	if err != nil {
		return nil, err
	}
	matcher := &namespaceMatcherImpl{namespaceRegex}
	return matcher.match, nil
}

type namespaceMatcherImpl struct {
	namespaceRegex *regexp.Regexp
}

func (p *namespaceMatcherImpl) match(name *v1.ImageName) []*v1.Alert_Violation {
	var namespace string
	remoteSplit := strings.Split(name.GetRemote(), "/")
	if len(remoteSplit) >= 2 {
		namespace = remoteSplit[0]
	}

	var violations []*v1.Alert_Violation
	if namespace != "" && p.namespaceRegex.MatchString(namespace) {
		v := &v1.Alert_Violation{
			Message: fmt.Sprintf("Image namspace matched: %s", p.namespaceRegex),
		}
		violations = append(violations, v)
	}
	return violations
}
