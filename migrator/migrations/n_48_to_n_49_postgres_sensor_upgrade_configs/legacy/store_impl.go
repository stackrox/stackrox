package bolt

import (
	"context"
	"time"

	proto "github.com/gogo/protobuf/proto"
	storage "github.com/stackrox/rox/generated/storage"
	metrics "github.com/stackrox/rox/migrator/migrations/postgreshelper/metrics"
	singletonstore "github.com/stackrox/rox/pkg/bolthelper/singletonstore"
	ops "github.com/stackrox/rox/pkg/metrics"
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
