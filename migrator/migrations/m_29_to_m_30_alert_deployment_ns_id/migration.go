package m29to30

import (
	"time"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

const (
	logInterval = 30 * time.Second
)

var (
	alertBucketName     = []byte("alerts\x00")
	namespaceBucketName = []byte("namespaces")
	migration           = types.Migration{
		StartingSeqNum: 29,
		VersionAfter:   storage.Version{SeqNum: 30},
		Run:            updateAlertDeploymentWithNamespaceID,
	}
)

type namespaceKey struct {
	clusterID, namespace string
}

func updateAlertDeploymentWithNamespaceID(boltDB *bolt.DB, badgerDB *badger.DB) error {
	namespaceKeyMap, err := getNamespaceKeyMappings(boltDB)
	if err != nil {
		return err
	}

	return updateAlertDeployments(badgerDB, namespaceKeyMap)
}

func getNamespaceKeyMappings(boltDB *bolt.DB) (map[namespaceKey]string, error) {
	namespaceKeyMap := make(map[namespaceKey]string)
	err := boltDB.View(func(tx *bolt.Tx) error {
		return tx.Bucket(namespaceBucketName).ForEach(func(k, v []byte) error {
			var namespace storage.NamespaceMetadata
			if err := proto.Unmarshal(v, &namespace); err != nil {
				return errors.Wrapf(err, "unmarshal error for namespace: %s", k)
			}

			namespaceKeyMap[namespaceKey{namespace.GetClusterId(), namespace.GetName()}] = namespace.GetId()
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return namespaceKeyMap, nil
}

func updateAlertDeployments(badgerDB *badger.DB, namespaceKeyMap map[namespaceKey]string) error {
	return badgerDB.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		batch := badgerDB.NewWriteBatch()
		defer batch.Cancel()

		lastLog := time.Now()
		count := 0
		for it.Seek(alertBucketName); it.ValidForPrefix(alertBucketName); it.Next() {
			if batch.Error() != nil {
				return batch.Error()
			}

			key := it.Item().KeyCopy(nil)
			err := it.Item().Value(func(v []byte) error {
				var alert storage.Alert
				if err := proto.Unmarshal(v, &alert); err != nil {
					return errors.Wrapf(err, "unmarshal error for alert: %s", key)
				}

				if alert.GetDeployment() == nil {
					return nil
				}

				nsID, ok := namespaceKeyMap[namespaceKey{alert.GetDeployment().GetClusterId(), alert.GetDeployment().GetNamespace()}]
				if !ok {
					return nil
				}

				alert.Deployment.NamespaceId = nsID
				data, err := proto.Marshal(&alert)
				if err != nil {
					return errors.Wrapf(err, "marshal error for alert: %s", key)
				}

				if err := batch.Set(key, data); err != nil {
					return errors.Wrapf(err, "error setting key/value in Badger for bucket %q", string(alertBucketName))
				}

				count++
				if time.Since(lastLog) > logInterval {
					log.WriteToStderrf("Successfully rewrote %d alerts", count)
					lastLog = time.Now()
				}

				return nil
			})
			if err != nil {
				return err
			}
		}
		if err := batch.Flush(); err != nil {
			return errors.Wrapf(err, "error flushing BadgerDB for bucket %q", string(alertBucketName))
		}
		log.WriteToStderrf("Successfully rewrote all %d alerts", count)
		return nil
	})
}

func init() {
	migrations.MustRegisterMigration(migration)
}
