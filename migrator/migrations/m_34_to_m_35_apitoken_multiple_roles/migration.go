package m33tom34

import (
	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var (
	migration = types.Migration{
		StartingSeqNum: 34,
		VersionAfter:   storage.Version{SeqNum: 35},
		Run: func(databases *types.Databases) error {
			err := migrateAPITokenInfo(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "updating dackbox graph schema")
			}
			return nil
		},
	}

	apiTokensBucket = []byte("apiTokens")
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func migrateAPITokenInfo(db *bbolt.DB) error {
	tokensToMigrate := make(map[string]*storage.TokenMetadata)
	// Remove the namespace sac keys from the cluster graph keys.
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(apiTokensBucket)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			tokenMD := &storage.TokenMetadata{}
			if err := proto.Unmarshal(v, tokenMD); err != nil {
				log.WriteToStderrf("Failed to unmarshal API token data for key %s: %v", k, err)
				return nil
			}
			if len(tokenMD.GetRoles()) != 0 || tokenMD.GetRole() == "" {
				return nil // already migrated
			}
			tokensToMigrate[string(k)] = tokenMD
			return nil
		})
	})
	if err != nil {
		return errors.Wrap(err, "reading API token data")
	}

	if len(tokensToMigrate) == 0 {
		return nil // nothing to do
	}

	for _, token := range tokensToMigrate {
		token.Roles = []string{token.Role}
	}

	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(apiTokensBucket)
		if bucket == nil {
			return errors.Errorf("bucket %s not found", apiTokensBucket)
		}
		for id, token := range tokensToMigrate {
			bytes, err := proto.Marshal(token)
			if err != nil {
				log.WriteToStderrf("failed to marshal migrated API token metadata for key %s: %v", id, err)
				continue
			}
			if err := bucket.Put([]byte(id), bytes); err != nil {
				return err
			}
		}
		return nil
	})
}
