package matcher

import (
	"regexp"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

const (
	// excludedOperatorNamespace defines the constant for openshift-operators which is a default namespace for many
	// third-party operators that we do *not* want to specify as system services
	excludedOperatorNamespace = "openshift-operators"
)

var systemNamespaceRegex = regexp.MustCompile(`^kube.|^openshift.*|^redhat.*|^istio-system$`)

type platformMatcherImpl struct {
}

func (p *platformMatcherImpl) MatchAlert(alert *storage.Alert) (bool, error) {
	if alert == nil {
		return false, errors.New("Error matching alert: alert must be non nil")
	}
	if alert.GetDeployment() == nil {
		return false, nil
	}
	return p.matchNamespace(alert.GetDeployment().GetNamespace()), nil
}

func (p *platformMatcherImpl) MatchDeployment(deployment *storage.Deployment) (bool, error) {
	if deployment == nil {
		return false, errors.New("Error matching deployment: deployment must be non nil")
	}
	return p.matchNamespace(deployment.GetNamespace()), nil
}

func (p *platformMatcherImpl) matchNamespace(namespace string) bool {
	if namespace == excludedOperatorNamespace {
		return false
	}
	return systemNamespaceRegex.MatchString(namespace)
}
