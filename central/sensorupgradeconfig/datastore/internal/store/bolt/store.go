package bolt

import (
	"context"
	"time"

	proto "github.com/gogo/protobuf/proto"
	metrics "github.com/stackrox/stackrox/central/metrics"
	storage "github.com/stackrox/stackrox/generated/storage"
	singletonstore "github.com/stackrox/stackrox/pkg/bolthelper/singletonstore"
	ops "github.com/stackrox/stackrox/pkg/metrics"
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
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "SensorUpgradeConfig")
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
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "SensorUpgradeConfig")
	return s.underlying.Upsert(sensorupgradeconfig)
}
