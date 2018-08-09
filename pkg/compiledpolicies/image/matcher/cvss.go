package matcher

import (
	"fmt"
	"math"

	"github.com/stackrox/rox/generated/api/v1"
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

func (p *cvssMatcherImpl) match(image *v1.Image) (violations []*v1.Alert_Violation) {
	minimum := float32(math.MaxFloat32)
	var maximum float32
	var average float32

	var numVulns float32
	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			minimum = min(minimum, vuln.GetCvss())
			maximum = max(maximum, vuln.GetCvss())
			average += vuln.GetCvss()
			numVulns++
		}
	}

	var value float32
	switch p.cvss.GetMathOp() {
	case v1.MathOP_MIN:
		// This case is necessary due to setting the minimum value as the largest float
		// If there are no vulns then the minimum value would be max float
		if numVulns > 0 {
			value = minimum
		}
	case v1.MathOP_MAX:
		value = maximum
	case v1.MathOP_AVG:
		if numVulns == 0 {
			value = 0
		} else {
			value = average / numVulns
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
	if comparatorFunc(value, p.cvss.GetValue()) {
		violations = append(violations, &v1.Alert_Violation{
			Message: fmt.Sprintf("The %s(cvss) = %v. %v is %v threshold of %v", p.cvss.GetMathOp(), value, value, comparatorChar, p.cvss.GetValue()),
		})
	}
	return
}

func min(x, y float32) float32 {
	return float32(math.Min(float64(x), float64(y)))
}

func max(x, y float32) float32 {
	return float32(math.Max(float64(x), float64(y)))
}
