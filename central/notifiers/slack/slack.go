package slack

import (
	"github.com/stackrox/rox/central/notifiers/metadatagetter"
	"github.com/stackrox/rox/generated/storage"
	mitreDS "github.com/stackrox/rox/pkg/mitre/datastore"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/notifiers/slack"
)

func init() {
	notifiers.Add(notifiers.SlackType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		s, err := slack.NewSlack(notifier, metadatagetter.Singleton(), mitreDS.Singleton())
		return s, err
	})
}
