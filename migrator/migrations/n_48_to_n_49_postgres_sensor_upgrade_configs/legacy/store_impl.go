// This file was originally generated with
// //go:generate cp ../../../../central/sensorupgradeconfig/datastore/internal/store/bolt/store.go store_impl.go

package legacy

import (
	"context"

	proto "github.com/gogo/protobuf/proto"
	storage "github.com/stackrox/rox/generated/storage"
	singletonstore "github.com/stackrox/rox/pkg/bolthelper/singletonstore"
	bbolt "go.etcd.io/bbolt"
)

var (
	bucketName = []byte("sensor-upgrade-config")
)

// New creates a new BoltDB store
func New(db *bbolt.DB) *store {
	return &store{underlying: singletonstore.New(db, bucketName, func() proto.Message {
		return new(storage.SensorUpgradeConfig)
	}, "SensorUpgradeConfig")}
}

type store struct {
	underlying singletonstore.SingletonStore
}

func (s *store) Get(_ context.Context) (*storage.SensorUpgradeConfig, bool, error) {
	msg, err := s.underlying.Get()
	if err != nil {
		return nil, false, err
	}
	if msg == nil {
		return nil, false, nil
	}
	return msg.(*storage.SensorUpgradeConfig), true, nil
}

func (s *store) Upsert(_ context.Context, sensorupgradeconfig *storage.SensorUpgradeConfig) error {
	return s.underlying.Upsert(sensorupgradeconfig)
}
