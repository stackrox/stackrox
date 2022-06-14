package m102tom103

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/binenc"
	"github.com/stackrox/rox/pkg/uuid"
	bolt "go.etcd.io/bbolt"
)

var (
	bucketName = []byte("groups2")

	migration = types.Migration{
		StartingSeqNum: 103,
		VersionAfter:   storage.Version{SeqNum: 104},
		Run: func(databases *types.Databases) error {
			if err := addIDToGroups(databases.BoltDB); err != nil {
				return errors.Wrap(err, "error adding ids to groups")
			}
			return nil
		},
	}
)

func addIDToGroups(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			// If we can successfully unmarshal the values to the group proto, we shall skip it.
			// We can safely ignore the error here and continue with the deserialization for old groups.
			if err := proto.Unmarshal(v, &storage.Group{}); err == nil {
				return nil
			}

			grp, err := deserialize(k, v)
			if err != nil {
				return err
			}

			if grp.GetProps().GetId() != "" {
				return nil
			}

			grp.GetProps().Id = generateGroupID()

			data, err := proto.Marshal(grp)
			if err != nil {
				return err
			}

			if err := bucket.Put([]byte(grp.GetProps().GetId()), data); err != nil {
				return err
			}

			// Delete the old entry within the bucket, avoiding storing groups twice.
			return bucket.Delete(k)
		})
	})
}

func init() {
	migrations.MustRegisterMigration(migration)
}

func deserialize(key, value []byte) (*storage.Group, error) {
	props, err := deserializePropsKey(key)
	if err != nil {
		return nil, err
	}

	return &storage.Group{
		Props:    props,
		RoleName: string(value),
	}, nil
}

func deserializePropsKey(key []byte) (*storage.GroupProperties, error) {
	parts, err := binenc.DecodeBytesList(key)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bytes list")
	}
	if len(parts) != 3 {
		return nil, errors.Errorf("decoded bytes list has %d elements, expected 3", len(parts))
	}

	if len(parts[0])+len(parts[1])+len(parts[2]) == 0 {
		return nil, nil
	}

	return &storage.GroupProperties{
		AuthProviderId: string(parts[0]),
		Key:            string(parts[1]),
		Value:          string(parts[2]),
	}, nil
}

func generateGroupID() string {
	return "io.stackrox.group." + uuid.NewV4().String()
}
