package jira

import (
	"github.com/stackrox/rox/generated/storage"
	mitreDataStore "github.com/stackrox/rox/pkg/mitre/datastore"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/notifiers/jira"
	"github.com/stackrox/rox/sensor/admission-control/notifiers/metadatagetter"
)

func init() {
	notifiers.Add("jira", func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		j, err := jira.NewJira(notifier, metadatagetter.Singleton(), mitreDataStore.Singleton())
		return j, err
	})
}
