package matcher

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

func init() {
	compilers = append(compilers, newDropCapMatcher)
}

func newDropCapMatcher(policy *storage.Policy) (Matcher, error) {
	fields := policy.GetFields()
	if len(fields.GetDropCapabilities()) == 0 {
		return nil, nil
	}

	dropCap := make(map[string]struct{})
	for _, cap := range fields.GetDropCapabilities() {
		dropCap[strings.ToUpper(cap)] = struct{}{}
	}
	matcher := &dropCapMatcherImpl{proto: policy, dropCap: dropCap}
	return matcher.match, nil
}

type dropCapMatcherImpl struct {
	proto *storage.Policy

	dropCap map[string]struct{}
}

func (p *dropCapMatcherImpl) match(security *storage.SecurityContext) []*storage.Alert_Violation {
	if security == nil {
		return nil
	}

	matchedCap := 0
	for _, cap := range security.GetDropCapabilities() {
		if _, ok := p.dropCap[strings.ToUpper(cap)]; ok {
			matchedCap++
		}
	}

	var violations []*storage.Alert_Violation
	if len(p.dropCap) == matchedCap {
		violations = append(violations, &storage.Alert_Violation{
			Message: fmt.Sprintf("SecurityContext with add capabilities %+v matches policy %+v", security.GetDropCapabilities(), p.proto.GetFields().GetDropCapabilities()),
		})
	}

	return violations
}
