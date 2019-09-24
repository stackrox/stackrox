package badgermigration

import (
	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/log"
)

const (
	batchSize = 2000
)

var (
	deploymentBucket     = []byte("deployments")
	listDeploymentBucket = []byte("deployments_list")

	alertBucket         = []byte("alerts")
	listAlertBucketName = []byte("alerts_list")

	imageBucket     = []byte("imageBucket")
	listImageBucket = []byte("images_list")

	processIndicatorBucket = []byte("process_indicators")
	uniqueProcessesBucket  = []byte("process_indicators_unique")
)

// RewriteData rewrites the core bolt data into badger
func RewriteData(db *bolt.DB, badgerDB *badger.DB) error {
	// Alert
	if err := rewrite(db, badgerDB, alertBucket); err != nil {
		return err
	}
	if err := rewrite(db, badgerDB, listAlertBucketName); err != nil {
		return err
	}

	// Deployment
	if err := rewrite(db, badgerDB, deploymentBucket); err != nil {
		return err
	}
	if err := rewrite(db, badgerDB, listDeploymentBucket); err != nil {
		return err
	}

	// Image
	if err := rewrite(db, badgerDB, imageBucket); err != nil {
		return err
	}
	if err := rewrite(db, badgerDB, listImageBucket); err != nil {
		return err
	}

	// Process Indicators
	if err := rewrite(db, badgerDB, processIndicatorBucket); err != nil {
		return err
	}
	if err := rewrite(db, badgerDB, uniqueProcessesBucket); err != nil {
		return err
	}
	return nil
}

func rewrite(db *bolt.DB, badgerDB *badger.DB, bucketName []byte) error {
	log.WriteToStderrf("Rewriting Bucket %q", string(bucketName))
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		if bucket == nil {
			return nil
		}
		totalKeys := bucket.Stats().KeyN
		log.WriteToStderrf("Total keys in bucket: %d", totalKeys)

		keysWritten := 0
		batch := badgerDB.NewWriteBatch()
		err := bucket.ForEach(func(k, v []byte) error {
			if batch.Error() != nil {
				return batch.Error()
			}
			key := make([]byte, 0, len(bucketName)+len(k)+1)
			key = append(key, bucketName...)
			// The separator is a null char
			key = append(key, []byte("\x00")...)
			key = append(key, k...)

			if err := batch.Set(key, v); err != nil {
				return errors.Wrapf(err, "error setting key/value in Badger for bucket %q", string(bucketName))
			}

			keysWritten++
			if keysWritten%batchSize == 0 {
				if err := batch.Flush(); err != nil {
					return err
				}
				log.WriteToStderrf("Written %d/%d keys for bucket %q", keysWritten, totalKeys, string(bucketName))
				batch = badgerDB.NewWriteBatch()
			}
			return nil
		})
		defer batch.Cancel()
		if err != nil {
			return err
		}
		log.WriteToStderrf("Running final flush for %s into BadgerDB", string(bucketName))
		if err := batch.Flush(); err != nil {
			return errors.Wrapf(err, "error flushing BadgerDB for bucket %q", string(bucketName))
		}
		log.WriteToStderrf("Wrote %s into BadgerDB. Deleting Bucket from Bolt", string(bucketName))
		if err := tx.DeleteBucket(bucketName); err != nil {
			return errors.Wrapf(err, "error deleting bucket %q from Bolt", string(bucketName))
		}
		log.WriteToStderrf("Successfully deleted bucket %q", string(bucketName))
		return nil
	})
}
