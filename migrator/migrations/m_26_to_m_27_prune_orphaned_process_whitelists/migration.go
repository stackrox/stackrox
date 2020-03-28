package m26to27

import (
	"fmt"
	"strings"
	"time"

	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	protoTypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

const (
	orphanWindow = 30 * time.Minute
	secondInt64  = int64(time.Second)
)

var (
	deploymentBucket = []byte("deployments\x00")
	whitelistBucket  = []byte("processWhitelists2")
	migration        = types.Migration{
		StartingSeqNum: 26,
		VersionAfter:   storage.Version{SeqNum: 27},
		Run:            pruneOrphanedProcessWhitelists,
	}
)

func pruneOrphanedProcessWhitelists(boltDB *bolt.DB, badgerDB *badger.DB) error {
	deploymentSet := make(map[string]struct{})
	err := badgerDB.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(deploymentBucket); it.ValidForPrefix(deploymentBucket); it.Next() {
			id := strings.TrimPrefix(string(it.Item().Key()), string(deploymentBucket))
			deploymentSet[id] = struct{}{}
		}

		return nil
	})

	if err != nil {
		return err
	}
	var whitelistKeysToRemove []string
	now := protoTypes.TimestampNow()
	err = boltDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(whitelistBucket)
		if b == nil {
			log.WriteToStderrf("Process whitelists bucket: %s does not exist", whitelistBucket)
			return nil
		}

		err := b.ForEach(func(k, v []byte) error {
			var whitelist storage.ProcessWhitelist
			if err := proto.Unmarshal(v, &whitelist); err != nil {
				log.WriteToStderr(fmt.Sprintf("Unmarshal error for whitelist: %s, %s\nerr: %s", k, v, err))
				return nil
			}

			if _, ok := deploymentSet[whitelist.GetKey().GetDeploymentId()]; !ok {
				if sub(now, whitelist.GetCreated()) < orphanWindow {
					return nil
				}

				whitelistKeysToRemove = append(whitelistKeysToRemove, whitelist.GetId())
			}

			return nil
		})

		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	err = boltDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(whitelistBucket)
		if b == nil {
			log.WriteToStderrf("Process whitelists bucket: %s does not exist", whitelistBucket)
			return nil
		}

		for _, id := range whitelistKeysToRemove {
			if err := b.Delete([]byte(id)); err != nil {
				log.WriteToStderr(fmt.Sprintf("Unable to delete process whitelist id: %s, err: %s", id, err))
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func sub(ts1, ts2 *protoTypes.Timestamp) time.Duration {
	if ts1 == nil || ts2 == nil {
		return 0
	}
	seconds := int64(ts1.GetSeconds() - ts2.GetSeconds())
	nanos := int64(ts1.GetNanos() - ts2.GetNanos())

	return time.Duration(int64(seconds)*secondInt64 + int64(nanos))
}

func init() {
	migrations.MustRegisterMigration(migration)
}
