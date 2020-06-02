package m36tom37

import (
	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var (
	migration = types.Migration{
		StartingSeqNum: 36,
		VersionAfter:   storage.Version{SeqNum: 37},
		Run: func(databases *types.Databases) error {
			err := addDefaultMCRIntegration(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "adding default MCR integration")
			}
			return nil
		},
	}

	imageIntegrationBucket = []byte("imageintegrations")

	// this image integration should match the one in the default integrations
	mcrIntegration = &storage.ImageIntegration{
		Id:         "4b36a1c3-2d6f-452e-a70f-6c388a0ff947",
		Name:       "Public Microsoft Container Registry",
		Type:       "docker",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "mcr.microsoft.com",
			},
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func addDefaultMCRIntegration(db *bbolt.DB) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(imageIntegrationBucket)
		if bucket == nil {
			return errors.New("image integrations bucket not found")
		}

		existingMCRIntegration := bucket.Get([]byte(mcrIntegration.Id))
		if existingMCRIntegration != nil {
			// Should we check for name of integration match here?
			return nil
		}

		i, err := proto.Marshal(mcrIntegration)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(mcrIntegration.Id), i)
	})
}
