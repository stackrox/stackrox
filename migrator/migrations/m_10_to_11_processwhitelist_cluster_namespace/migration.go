package m10to11

import (
	"fmt"

	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

const (
	deploymentContainerKeyPrefix = "DC"
)

var (
	listDeploymentBucketName = []byte("deployments_list")
	pWBucketName             = []byte("processWhitelists")
	pWResultsBucketName      = []byte("processWhitelistResults")
	newPWBucketName          = []byte("processWhitelists2")
)

type deploymentMeta struct {
	clusterID string
	namespace string
}

func retrieveDeployments(db *bolt.DB) (map[string]*deploymentMeta, error) {
	var deployments = make(map[string]*deploymentMeta)
	listDeploymentBucket := bolthelpers.TopLevelRef(db, listDeploymentBucketName)
	err := listDeploymentBucket.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(_, v []byte) error {
			var listDeployment storage.ListDeployment
			if err := proto.Unmarshal(v, &listDeployment); err != nil {
				return err
			}
			deployments[listDeployment.GetId()] = &deploymentMeta{
				clusterID: listDeployment.GetClusterId(),
				namespace: listDeployment.GetNamespace(),
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return deployments, nil
}

func updateProcessWhitelists(db *bolt.DB, deployments map[string]*deploymentMeta) error {
	err := db.Update(func(tx *bolt.Tx) error {
		pWBucket := tx.Bucket(pWBucketName)
		if pWBucket == nil {
			return nil
		}

		newPWBucket, err := tx.CreateBucketIfNotExists(newPWBucketName)
		if err != nil {
			return err
		}

		c := pWBucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var processWhitelist storage.ProcessWhitelist
			if err := proto.Unmarshal(v, &processWhitelist); err != nil {
				return err
			}
			deploymentMeta := deployments[processWhitelist.GetKey().GetDeploymentId()]
			if deploymentMeta == nil {
				if err := newPWBucket.Put(k, v); err != nil {
					return err
				}
				continue
			}

			processWhitelist.Key.ClusterId = deploymentMeta.clusterID
			processWhitelist.Key.Namespace = deploymentMeta.namespace
			id, err := keyToID(processWhitelist.GetKey())
			if err != nil {
				return err
			}

			pwBytes, err := proto.Marshal(&processWhitelist)
			if err != nil {
				return err
			}

			if err = newPWBucket.Put([]byte(id), pwBytes); err != nil {
				return err
			}
		}

		return tx.DeleteBucket(pWBucketName)
	})
	return err
}

func updateProcessWhitelistResults(db *bolt.DB, deployments map[string]*deploymentMeta) error {
	if exists, err := bolthelpers.BucketExists(db, pWResultsBucketName); err != nil {
		return err
	} else if !exists {
		return nil
	}

	pWResultsBucket := bolthelpers.TopLevelRef(db, pWResultsBucketName)
	err := pWResultsBucket.Update(func(b *bolt.Bucket) error {
		return b.ForEach(func(k, v []byte) error {
			var pWResults storage.ProcessWhitelistResults
			if err := proto.Unmarshal(v, &pWResults); err != nil {
				return err
			}
			deploymentMeta := deployments[pWResults.GetDeploymentId()]
			if deploymentMeta == nil {
				return nil
			}

			pWResults.ClusterId = deploymentMeta.clusterID
			pWResults.Namespace = deploymentMeta.namespace

			pWResultsBytes, err := proto.Marshal(&pWResults)
			if err != nil {
				return err
			}

			return b.Put(k, pWResultsBytes)
		})
	})

	return err
}

func keyToID(key *storage.ProcessWhitelistKey) (string, error) {
	if allNotEmpty(key.GetClusterId(), key.GetNamespace(), key.GetDeploymentId(), key.GetContainerName()) {
		return fmt.Sprintf("%s:%s:%s:%s:%s", deploymentContainerKeyPrefix, key.GetClusterId(), key.GetNamespace(), key.GetDeploymentId(), key.GetContainerName()), nil
	}
	return "", fmt.Errorf("invalid key %+v: doesn't match any of our known patterns", key)
}

func allNotEmpty(strs ...string) bool {
	for _, s := range strs {
		if s == "" {
			return false
		}
	}
	return true
}

var (
	migration = types.Migration{
		StartingSeqNum: 10,
		VersionAfter:   storage.Version{SeqNum: 11},
		Run: func(db *bolt.DB, _ *badger.DB) error {
			deployments, err := retrieveDeployments(db)
			if err != nil {
				return errors.Wrap(err, "retrieving deployments")
			}
			err = updateProcessWhitelists(db, deployments)
			if err != nil {
				return errors.Wrap(err, "updating processwhitelists")
			}
			err = updateProcessWhitelistResults(db, deployments)
			if err != nil {
				return errors.Wrap(err, "updating processwhitelistresults")
			}
			return nil
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}
