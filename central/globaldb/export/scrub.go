package export

import (
	bolt "github.com/etcd-io/bbolt"
	imageIntegrationStore "github.com/stackrox/rox/central/imageintegration/store"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/pkg/secrets"
)

func scrubSensitiveData(db *bolt.DB) error {
	if err := notifierDS.Scrub(db); err != nil {
		return err
	}

	return clearImageIntegrationConfigs(db)
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
