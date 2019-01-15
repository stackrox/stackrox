package store

import (
	"fmt"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dberrors"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/secondarykey"
	"github.com/stackrox/rox/pkg/uuid"
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

// GetImageIntegration returns integration with given id.
func (b *storeImpl) GetImageIntegration(id string) (integration *storage.ImageIntegration, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "ImageIntegration")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(imageIntegrationBucket)
		integration, exists, err = b.getImageIntegration(id, bucket)
		return err
	})
	return
}

// GetImageIntegrations retrieves integrations from bolt
func (b *storeImpl) GetImageIntegrations() ([]*storage.ImageIntegration, error) {
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

// AddImageIntegration adds a integration into bolt
func (b *storeImpl) AddImageIntegration(integration *storage.ImageIntegration) (string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "ImageIntegration")
	integration.Id = uuid.NewV4().String()
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(imageIntegrationBucket)
		_, exists, err := b.getImageIntegration(integration.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Image integration %s (%s) cannot be added because it already exists", integration.GetId(), integration.GetName())
		}
		if err := secondarykey.CheckUniqueKeyExistsAndInsert(tx, imageIntegrationBucket, integration.GetId(), integration.GetName()); err != nil {
			return fmt.Errorf("Could not add image integration due to name validation: %s", err)
		}
		bytes, err := proto.Marshal(integration)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(integration.GetId()), bytes)
	})
	return integration.Id, err
}

// UpdateImageIntegration upserts a integration into bolt
func (b *storeImpl) UpdateImageIntegration(integration *storage.ImageIntegration) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "ImageIntegration")
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(imageIntegrationBucket)
		// If the update is changing the name, check if the name has already been taken
		if val, _ := secondarykey.GetCurrentUniqueKey(tx, imageIntegrationBucket, integration.GetId()); val != integration.GetName() {
			if err := secondarykey.UpdateUniqueKey(tx, imageIntegrationBucket, integration.GetId(), integration.GetName()); err != nil {
				return fmt.Errorf("Could not update integration due to name validation: %s", err)
			}
		}
		bytes, err := proto.Marshal(integration)
		if err != nil {
			return err
		}
		return b.Put([]byte(integration.GetId()), bytes)
	})
}

// RemoveImageIntegration removes a integration from bolt
func (b *storeImpl) RemoveImageIntegration(id string) error {
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
