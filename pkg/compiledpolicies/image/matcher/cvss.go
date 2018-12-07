package matcher

import (
	"fmt"
	"math"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

func init() {
	compilers = append(compilers, newCVSSMatcher)
}

func newCVSSMatcher(policy *v1.Policy) (Matcher, error) {
	cvss := policy.GetFields().GetCvss()
	if cvss == nil {
		return nil, nil
	}
	matcher := &cvssMatcherImpl{cvss: cvss}
	return matcher.match, nil
}

type cvssMatcherImpl struct {
	cvss *v1.NumericalPolicy
}

func (p *cvssMatcherImpl) match(image *storage.Image) (violations []*v1.Alert_Violation) {
	var maxCVSS float32

	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			maxCVSS = max(maxCVSS, vuln.GetCvss())
		}
	}

	var comparatorFunc func(x, y float32) bool
	var comparatorChar string
	switch p.cvss.GetOp() {
	case v1.Comparator_LESS_THAN:
		comparatorFunc = func(x, y float32) bool { return x < y }
		comparatorChar = "<"
	case v1.Comparator_LESS_THAN_OR_EQUALS:
		comparatorFunc = func(x, y float32) bool { return x <= y }
		comparatorChar = "<="
	case v1.Comparator_EQUALS:
		comparatorFunc = func(x, y float32) bool { return x == y }
		comparatorChar = "="
	case v1.Comparator_GREATER_THAN_OR_EQUALS:
		comparatorFunc = func(x, y float32) bool { return x >= y }
		comparatorChar = ">="
	case v1.Comparator_GREATER_THAN:
		comparatorFunc = func(x, y float32) bool { return x > y }
		comparatorChar = ">"
	}
	if comparatorFunc(maxCVSS, p.cvss.GetValue()) {
		violations = append(violations, &v1.Alert_Violation{
			Message: fmt.Sprintf("Max CVSS = %v, which is %v threshold of %v", maxCVSS, comparatorChar, p.cvss.GetValue()),
		})
	}
	return
}

func max(x, y float32) float32 {
	return float32(math.Max(float64(x), float64(y)))
}
