package matcher

import (
	"regexp"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

type platformMatcherImpl struct {
	regexes []*regexp.Regexp
}

func (p *platformMatcherImpl) SetRegexes(regexes []*regexp.Regexp) {
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
	return p.matchNamespace(deployment.GetNamespace()), nil
}

func (p *platformMatcherImpl) matchNamespace(namespace string) bool {
	for _, rule := range p.regexes {
		if rule.MatchString(namespace) {
			return true
		}
	}
	return false
}
