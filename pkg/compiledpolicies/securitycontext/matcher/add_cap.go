package matcher

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

func init() {
	compilers = append(compilers, newAddCapMatcher)
}

func newAddCapMatcher(policy *storage.Policy) (Matcher, error) {
	fields := policy.GetFields()
	if len(fields.GetAddCapabilities()) == 0 {
		return nil, nil
	}

	addCap := make(map[string]struct{})
	for _, cap := range fields.GetAddCapabilities() {
		addCap[strings.ToUpper(cap)] = struct{}{}
	}
	matcher := &addCapMatcherImpl{proto: policy, addCap: addCap}
	return matcher.match, nil
}

type addCapMatcherImpl struct {
	proto *storage.Policy

	addCap map[string]struct{}
}

func (p *addCapMatcherImpl) match(security *storage.SecurityContext) []*storage.Alert_Violation {
	if security == nil {
		return nil
	}

	matchedCap := 0
	for _, cap := range security.GetAddCapabilities() {
		if _, ok := p.addCap[strings.ToUpper(cap)]; ok {
			matchedCap++
		}
	}

	var violations []*storage.Alert_Violation
	if len(p.addCap) == matchedCap {
		violations = append(violations, &storage.Alert_Violation{
			Message: fmt.Sprintf("Container with add capabilities %+v matches policy %+v", security.GetAddCapabilities(), p.proto.GetFields().GetAddCapabilities()),
		})
	}

	return violations
}
