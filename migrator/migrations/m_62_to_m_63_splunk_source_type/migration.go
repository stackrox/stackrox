package m62tom63

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var (
	notifiersBucket = []byte("notifiers")

	sourceTypeMap = map[string]string{
		"alert": "stackrox-alert",
		"audit": "stackrox-audit-message",
	}
	jsonSourceType = "_json"

	migration = types.Migration{
		StartingSeqNum: 62,
		VersionAfter:   &storage.Version{SeqNum: 63},
		Run: func(databases *types.Databases) error {
			return migrateSplunkSourceType(databases.BoltDB)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func migrateSplunkSourceType(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(notifiersBucket)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			var notifier storage.Notifier
			if err := proto.Unmarshal(v, &notifier); err != nil {
				return err
			}
			if notifier.GetType() != "splunk" {
				return nil
			}
			splunk := notifier.GetSplunk()
			splunk.SourceTypes = make(map[string]string)
			if splunk.GetDerivedSourceType() {
				splunk.SourceTypes = sourceTypeMap
			} else {
				for k := range sourceTypeMap {
					splunk.SourceTypes[k] = jsonSourceType
				}
			}
			splunk.DerivedSourceTypeDeprecated = nil
			newData, err := proto.Marshal(&notifier)
			if err != nil {
				return err
			}
			return bucket.Put(k, newData)
		})
	})
}
