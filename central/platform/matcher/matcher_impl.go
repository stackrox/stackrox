package matcher

import (
	"regexp"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

var systemNamespaceRegex = regexp.MustCompile(`^kube-.*|^openshift-.*|^stackrox$|^rhacs-operator$|^open-cluster-management$|^multicluster-engine$|^aap$|^hive$|^nvidia-gpu-operator$`)

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
	return systemNamespaceRegex.MatchString(namespace)
}
