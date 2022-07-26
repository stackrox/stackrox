package bolt

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper/singletonstore"
	"go.etcd.io/bbolt"
)

var (
	bucketName = []byte("config")
)

// New creates a new BoltDB
func New(db *bbolt.DB) Store {
	return &store{underlying: singletonstore.New(db, bucketName, func() proto.Message {
		return new(storage.Config)
	}, "Config")}
}

type store struct {
	underlying singletonstore.SingletonStore
}

func (s *store) Get(_ context.Context) (*storage.Config, bool, error) {
	msg, err := s.underlying.Get()
	if err != nil {
		return nil, false, err
	}
	if msg == nil {
		return nil, false, nil
	}
	return msg.(*storage.Config), true, nil
}

func (s *store) Upsert(_ context.Context, config *storage.Config) error {
	return s.underlying.Upsert(config)
}
