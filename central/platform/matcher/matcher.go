package matcher

import (
	"regexp"

	"github.com/stackrox/rox/generated/storage"
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

func New() PlatformMatcher {
	return &platformMatcherImpl{}
}
