package m18to19

import (
	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/features"
)

var migration = types.Migration{
	StartingSeqNum: 18,
	VersionAfter:   storage.Version{SeqNum: 19},
	Run: func(db *bolt.DB, badgerDB *badger.DB) error {
		if !features.BadgerDB.Enabled() {
			return nil
		}
		return rewriteData(db, badgerDB)
	},
}

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

func init() {
	migrations.MustRegisterMigration(migration)
}

func rewriteResource(v []byte, msg proto.Message) ([]byte, error) {
	if err := proto.Unmarshal(v, msg); err != nil {
		return nil, err
	}
	return proto.Marshal(msg)
}

func rewriteAlert(v []byte) ([]byte, error) {
	var alert storage.Alert
	if err := proto.Unmarshal(v, &alert); err != nil {
		return nil, err
	}
	if len(alert.GetProcessViolation().GetProcesses()) > 40 {
		alert.ProcessViolation.Processes = alert.ProcessViolation.Processes[:40]
	}
	return proto.Marshal(&alert)
}

func rewriteDeployment(v []byte) ([]byte, error) {
	var deployment storage.Deployment
	return rewriteResource(v, &deployment)
}

func rewriteImage(v []byte) ([]byte, error) {
	var image storage.Image
	return rewriteResource(v, &image)
}

func rewriteData(db *bolt.DB, badgerDB *badger.DB) error {
	// Alert
	if err := rewrite(db, badgerDB, alertBucket, rewriteAlert); err != nil {
		return err
	}
	if err := rewrite(db, badgerDB, listAlertBucketName, nil); err != nil {
		return err
	}

	// Deployment
	if err := rewrite(db, badgerDB, deploymentBucket, rewriteDeployment); err != nil {
		return err
	}
	if err := rewrite(db, badgerDB, listDeploymentBucket, nil); err != nil {
		return err
	}

	// Image
	if err := rewrite(db, badgerDB, imageBucket, rewriteImage); err != nil {
		return err
	}
	if err := rewrite(db, badgerDB, listImageBucket, nil); err != nil {
		return err
	}

	// Process Indicators
	if err := rewrite(db, badgerDB, processIndicatorBucket, nil); err != nil {
		return err
	}
	if err := rewrite(db, badgerDB, uniqueProcessesBucket, nil); err != nil {
		return err
	}
	return nil
}

func rewrite(db *bolt.DB, badgerDB *badger.DB, bucketName []byte, fn func(v []byte) ([]byte, error)) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)
		if bucket == nil {
			return nil
		}

		badgerTxn := badgerDB.NewTransaction(true)
		err := bucket.ForEach(func(k, v []byte) error {
			if fn != nil {
				var err error
				if v, err = fn(v); err != nil {
					return errors.Wrap(err, "error executing fn in rewrite")
				}
			}

			key := append(bucketName, ':')
			key = append(key, k...)

			if err := badgerTxn.Set(key, v); err != nil && err != badger.ErrTxnTooBig {
				badgerTxn.Discard()
				return errors.Wrapf(err, "error setting key/value for bucket %q", string(bucketName))
			} else if err == badger.ErrTxnTooBig {
				if err := badgerTxn.Commit(); err != nil {
					badgerTxn.Discard()
					return errors.Wrapf(err, "error committing badger txn for bucket %q", string(bucketName))
				}
				badgerTxn.Discard()

				badgerTxn = badgerDB.NewTransaction(true)
				if err := badgerTxn.Set(key, v); err != nil {
					badgerTxn.Discard()
					return errors.Wrapf(err, "error setting key/value for bucket %q", string(bucketName))
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		if err := badgerTxn.Commit(); err != nil {
			return err
		}
		return tx.DeleteBucket(bucketName)
	})
}
