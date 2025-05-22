package matcher

import (
	"context"
	"regexp"

	configDS "github.com/stackrox/rox/central/config/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

// PlatformMatcher matches alerts and deployments against platform rules
//
//go:generate mockgen-wrapper
type PlatformMatcher interface {
	// MatchAlert returns true if the given alert matches platform rules
	MatchAlert(alert *storage.Alert) (bool, error)
	// MatchDeployment returns true if the given deployment matches platform rules
	MatchDeployment(deployment *storage.Deployment) (bool, error)
	SetRegexes(regexes []*regexp.Regexp)
}

func New(configDatastore configDS.DataStore) PlatformMatcher {
	allAccessCtx := sac.WithAllAccess(context.Background())
	regexes := []*regexp.Regexp{}
	config, _, _ := configDatastore.GetPlatformComponentConfig(allAccessCtx)
	for _, rule := range config.GetRules() {
		regex, _ := regexp.Compile(rule.GetNamespaceRule().GetRegex())
		regexes = append(regexes, regex)
	}
	return &platformMatcherImpl{
		regexes: regexes,
	}
}
