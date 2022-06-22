package bolt

import (
	"context"
	"time"

	proto "github.com/gogo/protobuf/proto"
	metrics "github.com/stackrox/rox/central/metrics"
	storage "github.com/stackrox/rox/generated/storage"
	singletonstore "github.com/stackrox/rox/pkg/bolthelper/singletonstore"
	ops "github.com/stackrox/rox/pkg/metrics"
	bbolt "go.etcd.io/bbolt"
)

var (
	bucketName = []byte("config")
)

// New creates a new BoltDB
func New(db *bbolt.DB) *Store {
	return &Store{underlying: singletonstore.New(db, bucketName, func() proto.Message {
		return new(storage.Config)
	}, "Config")}
}

type Store struct {
	underlying singletonstore.SingletonStore
}

func (s *Store) Get(_ context.Context) (*storage.Config, bool, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Config")
	msg, err := s.underlying.Get()
	if err != nil {
		return nil, false, err
	}
	if msg == nil {
		return nil, false, nil
	}
	return msg.(*storage.Config), true, nil
}

func (s *Store) Upsert(_ context.Context, config *storage.Config) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "Config")
	return s.underlying.Upsert(config)
}
