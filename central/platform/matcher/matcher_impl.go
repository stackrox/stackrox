package matcher

import (
	"context"
	"regexp"

	"github.com/pkg/errors"
	configDatastore "github.com/stackrox/rox/central/config/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

type platformMatcherImpl struct{}

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
	// This isn't the most compute efficient way of doing this, but realistically there aren't going to be an
	// insane number of platform component rules, so this is probably an okay way of doing this, and if we had something
	// in the datastore that updated a field on the platformMatcherImpl, it would cause a circular dependency.
	config, _, _ := configDatastore.Singleton().GetPlatformComponentConfig(sac.WithAllAccess(context.Background()))
	for _, rule := range config.Rules {
		if regexp.MustCompile(rule.GetNamespaceRule().Regex).MatchString(namespace) {
			return true
		}
	}
	return false
}
