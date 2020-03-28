package m15to16

import (
	"encoding/binary"

	"github.com/cloudflare/cfssl/log"
	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/batcher"
)

var migration = types.Migration{
	StartingSeqNum: 15,
	VersionAfter:   storage.Version{SeqNum: 16},
	Run:            updateListAlerts,
}

var (
	alertBucket     = []byte("alerts")
	alertListBucket = []byte("alerts_list")

	transactionsBucket = []byte("transactions")
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func updateListAlerts(db *bolt.DB, _ *badger.DB) error {
	idSet := make(map[string]struct{})
	var alertBucketSize, listAlertBucketSize int

	var listAlertsToRemove []string
	err := db.View(func(tx *bolt.Tx) error {
		alertBucket := tx.Bucket(alertBucket)
		listAlertBucket := tx.Bucket(alertListBucket)

		alertBucketSize = alertBucket.Stats().KeyN
		listAlertBucketSize = listAlertBucket.Stats().KeyN
		if alertBucketSize == listAlertBucketSize {
			return nil
		}
		err := alertBucket.ForEach(func(k, v []byte) error {
			idSet[string(v)] = struct{}{}
			return nil
		})
		if err != nil {
			return err
		}

		listAlertsToRemove = make([]string, 0, listAlertBucketSize-alertBucketSize)
		err = db.View(func(tx *bolt.Tx) error {
			listAlerts := tx.Bucket(alertListBucket)
			return listAlerts.ForEach(func(k, v []byte) error {
				if _, ok := idSet[string(k)]; !ok {
					listAlertsToRemove = append(listAlertsToRemove, string(k))
				}
				return nil
			})
		})
		return err
	})
	if err != nil {
		return err
	}

	batch := batcher.New(len(listAlertsToRemove), 1000)
	for start, end, valid := batch.Next(); valid; start, end, valid = batch.Next() {
		err := db.Update(func(tx *bolt.Tx) error {
			listAlerts := tx.Bucket(alertListBucket)
			for _, id := range listAlertsToRemove[start:end] {
				if err := listAlerts.Delete([]byte(id)); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		log.Infof("Successfully deleted %d/%d orphaned list alerts", end, len(listAlertsToRemove))
	}

	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(transactionsBucket)
		if bucket == nil {
			return nil
		}
		// Zero out the txn bucket so the indexer is forced to sync based on that
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, 0)

		return bucket.Put(alertBucket, b)
	})
}
