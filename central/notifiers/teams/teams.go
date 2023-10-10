package teams

import (
	"github.com/stackrox/rox/central/notifiers/metadatagetter"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/notifiers/teams"
)

func init() {
	notifiers.Add(notifiers.TeamsType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		s, err := teams.NewTeams(notifier, metadatagetter.Singleton())
		return s, err
	})
}
