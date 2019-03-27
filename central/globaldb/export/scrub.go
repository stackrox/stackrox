package export

import (
	bolt "github.com/etcd-io/bbolt"
	imageIntegrationStore "github.com/stackrox/rox/central/imageintegration/store"
	notifierStore "github.com/stackrox/rox/central/notifier/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/secrets"
)

func scrubSensitiveData(db *bolt.DB) error {
	if err := clearNotifierConfigs(db); err != nil {
		return err
	}

	return clearImageIntegrationConfigs(db)
}

func clearNotifierConfigs(db *bolt.DB) error {
	store := notifierStore.New(db)

	notifiers, err := store.GetNotifiers(&v1.GetNotifiersRequest{})
	if err != nil {
		return err
	}
	for _, n := range notifiers {
		n.Config = nil
		if err := store.UpdateNotifier(n); err != nil {
			return err
		}
	}
	return nil
}

func clearImageIntegrationConfigs(db *bolt.DB) error {
	store := imageIntegrationStore.New(db)

	integrations, err := store.GetImageIntegrations()
	if err != nil {
		return err
	}
	for _, d := range integrations {
		secrets.ScrubSecretsFromStruct(d)
		if err := store.UpdateImageIntegration(d); err != nil {
			return err
		}
	}
	return nil
}
