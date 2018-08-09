package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
)

func init() {
	compilers = append(compilers, newCVEMatcher)
}

func newCVEMatcher(policy *v1.Policy) (Matcher, error) {
	cve := policy.GetFields().GetCve()
	if cve == "" {
		return nil, nil
	}

	cveRegex, err := utils.CompileStringRegex(cve)
	if err != nil {
		return nil, err
	}
	matcher := &cveMatcherImpl{cveRegex}
	return matcher.match, nil
}

type cveMatcherImpl struct {
	cveRegex *regexp.Regexp
}

func (p *cveMatcherImpl) match(image *v1.Image) (violations []*v1.Alert_Violation) {
	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			if p.cveRegex.MatchString(vuln.GetCve()) {
				violations = append(violations, &v1.Alert_Violation{
					Message: fmt.Sprintf("'%v' in Component '%v' matches the regex '%+v'", vuln.GetCve(), component.GetName(), p.cveRegex),
					Link:    vuln.GetLink(),
				})
			}
		}
	}
	return
}
