package store

import (
	"errors"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globaldb/ops"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dberrors"
	"github.com/stackrox/rox/pkg/uuid"
)

type storeImpl struct {
	*bolt.DB
}

// GetDNRIntegration retrieves a DNR integration from Bolt.
func (b *storeImpl) GetDNRIntegration(id string) (integration *v1.DNRIntegration, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "DNRIntegration")
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(dnrIntegrationBucket))
		key := []byte(id)
		bytes := b.Get(key)
		if bytes == nil {
			return nil
		}
		exists = true
		integration = new(v1.DNRIntegration)
		err := proto.Unmarshal(bytes, integration)
		if err != nil {
			return fmt.Errorf("proto unmarshalling: %s", err)
		}
		return nil
	})
	if err != nil {
		err = fmt.Errorf("DNR integration retrieval: %s", err)
	}
	return
}

// GetDNRIntegrations retrieves all D&R integrations from bolt
func (b *storeImpl) GetDNRIntegrations(req *v1.GetDNRIntegrationsRequest) (integrations []*v1.DNRIntegration, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "DNRIntegration")
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(dnrIntegrationBucket))
		return b.ForEach(func(k, v []byte) error {
			var integration v1.DNRIntegration
			if err := proto.Unmarshal(v, &integration); err != nil {
				return fmt.Errorf("proto unmarshalling: %s", err)
			}
			// If a cluster id is provided, then only return integrations that match it.
			if req.GetClusterId() != "" {
				found := false
				for _, clusterID := range integration.GetClusterIds() {
					if clusterID == req.GetClusterId() {
						found = true
						break
					}
				}
				if !found {
					return nil
				}
			}
			integrations = append(integrations, &integration)
			return nil
		})
	})
	if err != nil {
		err = fmt.Errorf("DNR integration retrieval: %s", err)
	}
	return
}

// AddDNRIntegration adds a DNR integration to Bolt.
func (b *storeImpl) AddDNRIntegration(integration *v1.DNRIntegration) (string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "DNRIntegration")
	id := uuid.NewV4().String()
	integration.Id = id
	err := b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(dnrIntegrationBucket))
		key := []byte(id)
		bytes, err := proto.Marshal(integration)
		if err != nil {
			return fmt.Errorf("proto marshalling: %s", err)
		}
		return b.Put(key, bytes)
	})
	if err != nil {
		return "", fmt.Errorf("DNR integration insertion: %s", err)
	}
	return id, nil
}

// UpdateDNRIntegration updates the DNR integration in Bolt.
func (b *storeImpl) UpdateDNRIntegration(integration *v1.DNRIntegration) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "DNRIntegration")
	if integration.GetId() == "" {
		return errors.New("cannot update; empty id provided")
	}
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(dnrIntegrationBucket))
		key := []byte(integration.GetId())
		if b.Get(key) == nil {
			return dberrors.ErrNotFound{Type: "DNRIntegration"}
		}
		bytes, err := proto.Marshal(integration)
		if err != nil {
			return fmt.Errorf("DNR integration proto marshalling: %s", err)
		}
		return b.Put(key, bytes)
	})
}

// RemoveDNRIntegration removes the DNR integration from Bolt.
func (b *storeImpl) RemoveDNRIntegration(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "DNRIntegration")
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(dnrIntegrationBucket))
		key := []byte(id)
		if b.Get(key) == nil {
			return dberrors.ErrNotFound{Type: "DNRIntegration"}
		}
		return b.Delete(key)
	})
}
