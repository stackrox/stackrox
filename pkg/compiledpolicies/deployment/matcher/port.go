package matcher

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

func init() {
	compilers = append(compilers, newPortMatcher)
}

func newPortMatcher(policy *storage.Policy) (Matcher, error) {
	portPolicy := policy.GetFields().GetPortPolicy()
	if portPolicy == nil {
		return nil, nil
	}

	matcher := &portMatcherImpl{portPolicy: portPolicy}
	return matcher.match, nil
}

type portMatcherImpl struct {
	portPolicy *storage.PortPolicy
}

func (p *portMatcherImpl) match(deployment *storage.Deployment) []*storage.Alert_Violation {
	ports := deployment.GetPorts()
	var violations []*storage.Alert_Violation
	for _, port := range ports {
		violations = append(violations, p.matchPort(port)...)
	}
	return violations
}

func (p *portMatcherImpl) matchPort(port *storage.PortConfig) []*storage.Alert_Violation {
	if p.portPolicy.GetPort() != 0 && p.portPolicy.GetPort() != port.GetContainerPort() {
		return nil
	}

	if p.portPolicy.GetProtocol() != "" && !strings.EqualFold(p.portPolicy.GetProtocol(), port.GetProtocol()) {
		return nil
	}

	var violations []*storage.Alert_Violation
	violations = append(violations, &storage.Alert_Violation{
		Message: fmt.Sprintf("Port %+v matched configured policy %s", port, p.portPolicy),
	})
	return violations
}
