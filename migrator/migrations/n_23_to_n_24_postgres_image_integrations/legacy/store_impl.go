package bolt

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/postgresmigrationhelper/metrics"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/dberrors"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/secondarykey"
	bolt "go.etcd.io/bbolt"
)

type storeImpl struct {
	*bolt.DB
}

func (b *storeImpl) getImageIntegration(id string, bucket *bolt.Bucket) (integration *storage.ImageIntegration, exists bool, err error) {
	integration = new(storage.ImageIntegration)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, integration)
	return
}

// Get returns integration with given id.
func (b *storeImpl) Get(_ context.Context, id string) (integration *storage.ImageIntegration, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "ImageIntegration")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(imageIntegrationBucket)
		integration, exists, err = b.getImageIntegration(id, bucket)
		return err
	})
	return
}

// GetAll retrieves integrations from bolt
func (b *storeImpl) GetAll(_ context.Context) ([]*storage.ImageIntegration, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "ImageIntegration")
	var integrations []*storage.ImageIntegration
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(imageIntegrationBucket)
		return b.ForEach(func(k, v []byte) error {
			var integration storage.ImageIntegration
			if err := proto.Unmarshal(v, &integration); err != nil {
				return err
			}
			integrations = append(integrations, &integration)
			return nil
		})
	})
	return integrations, err
}

func addUniqueCheck(tx *bolt.Tx, integration *storage.ImageIntegration) error {
	if err := secondarykey.CheckUniqueKeyExistsAndInsert(tx, imageIntegrationBucket, integration.GetId(), integration.GetName()); err != nil {
		return errors.Wrap(err, "Could not add integration due to name validation")
	}
	return nil
}

func updateUniqueCheck(tx *bolt.Tx, integration *storage.ImageIntegration) error {
	if val, _ := secondarykey.GetCurrentUniqueKey(tx, imageIntegrationBucket, integration.GetId()); val != integration.GetName() {
		if err := secondarykey.UpdateUniqueKey(tx, imageIntegrationBucket, integration.GetId(), integration.GetName()); err != nil {
			return errors.Wrap(err, "Could not update integration due to name validation")
		}
	}
	return nil
}

// Upsert upserts an integration into bolt
func (b *storeImpl) Upsert(_ context.Context, integration *storage.ImageIntegration) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "ImageIntegration")

	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(imageIntegrationBucket)
		if bolthelper.Exists(bucket, integration.GetId()) {
			if err := updateUniqueCheck(tx, integration); err != nil {
				return err
			}
		} else {
			if err := addUniqueCheck(tx, integration); err != nil {
				return err
			}
		}
		bytes, err := proto.Marshal(integration)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(integration.GetId()), bytes)
	})
	return err
}

// Delete removes an integration from bolt
func (b *storeImpl) Delete(_ context.Context, id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "ImageIntegration")
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(imageIntegrationBucket)
		key := []byte(id)
		if exists := b.Get(key) != nil; !exists {
			return dberrors.ErrNotFound{Type: "ImageIntegration", ID: string(key)}
		}
		if err := secondarykey.RemoveUniqueKey(tx, imageIntegrationBucket, id); err != nil {
			return err
		}
		return b.Delete(key)
	})
}
