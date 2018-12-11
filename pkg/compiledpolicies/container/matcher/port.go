package matcher

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

func init() {
	compilers = append(compilers, newPortMatcher)
}

func newPortMatcher(policy *v1.Policy) (Matcher, error) {
	portPolicy := policy.GetFields().GetPortPolicy()
	if portPolicy == nil {
		return nil, nil
	}

	matcher := &portMatcherImpl{portPolicy: portPolicy}
	return matcher.match, nil
}

type portMatcherImpl struct {
	portPolicy *v1.PortPolicy
}

func (p *portMatcherImpl) match(container *storage.Container) []*v1.Alert_Violation {
	ports := container.GetPorts()
	var violations []*v1.Alert_Violation
	for _, port := range ports {
		violations = append(violations, p.matchPort(port)...)
	}
	return violations
}

func (p *portMatcherImpl) matchPort(port *storage.PortConfig) []*v1.Alert_Violation {
	if p.portPolicy.GetPort() != 0 && p.portPolicy.GetPort() != port.GetContainerPort() {
		return nil
	}

	if p.portPolicy.GetProtocol() != "" && !strings.EqualFold(p.portPolicy.GetProtocol(), port.GetProtocol()) {
		return nil
	}

	var violations []*v1.Alert_Violation
	violations = append(violations, &v1.Alert_Violation{
		Message: fmt.Sprintf("Port %+v matched configured policy %s", port, p.portPolicy),
	})
	return violations
}
