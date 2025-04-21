package matcher

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
)

var systemNamespaceRegex = regexp.MustCompile(`^kube-.*|^openshift-.*|^stackrox$|^rhacs-operator$|^open-cluster-management$|^multicluster-engine$|^aap$|^hive$|^nvidia-gpu-operator$`)

type platformMatcherImpl struct {
	regexes []*regexp.Regexp
}

func (p *platformMatcherImpl) SetRegexes(regexes []*regexp.Regexp) {
	fmt.Printf("Regexes updated: %v\n", regexes)
	p.regexes = regexes
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
	fmt.Printf("Matching deployment %s\n", deployment.GetName())
	return p.matchNamespace(deployment.GetNamespace()), nil
}

func (p *platformMatcherImpl) matchNamespace(namespace string) bool {
	if features.CustomizablePlatformComponents.Enabled() {
		for _, rule := range p.regexes {
			if rule.MatchString(namespace) {
				return true
			}
		}
		return false
	}
	return systemNamespaceRegex.MatchString(namespace)
}
