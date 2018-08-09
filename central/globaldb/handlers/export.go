package handlers

import (
	"os"

	"github.com/boltdb/bolt"
	imageIntegrationStore "github.com/stackrox/rox/central/imageintegration/store"
	notifierStore "github.com/stackrox/rox/central/notifier/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
)

// Export returns a compacted backup of the database.
func export(db *bolt.DB, exportedFilepath, compactedFilepath string) (*bolt.DB, error) {
	defer os.Remove(exportedFilepath)

	// This will block all other transactions until this has completed. We could use View for a hot backup
	err := db.Update(func(tx *bolt.Tx) error {
		return tx.CopyFile(exportedFilepath, 0600)
	})
	if err != nil {
		return nil, err
	}

	exportDB, err := bolthelper.New(exportedFilepath)
	if err != nil {
		return nil, err
	}
	defer exportDB.Close()

	clearNotifierConfigs(exportDB)
	if err != nil {
		return nil, err
	}

	clearImageIntegrationConfigs(exportDB)
	if err != nil {
		return nil, err
	}

	if err := exportDB.Sync(); err != nil {
		return nil, err
	}

	// Create completely clean DB and compact to it, wiping the secrets from cached memory
	newDB, err := bolt.Open(compactedFilepath, 0600, nil)
	if err != nil {
		return nil, err
	}
	if err := bolthelper.Compact(newDB, exportDB); err != nil {
		return nil, err
	}
	// Close the databases
	exportDB.Close()
	return newDB, nil
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

	integrations, err := store.GetImageIntegrations(&v1.GetImageIntegrationsRequest{})
	if err != nil {
		return err
	}
	for _, d := range integrations {
		d.Config = nil
		if err := store.UpdateImageIntegration(d); err != nil {
			return err
		}
	}
	return nil
}
