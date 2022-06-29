package bolt

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/postgresmigrationhelper/metrics"
	"github.com/stackrox/rox/pkg/bolthelper/singletonstore"
	ops "github.com/stackrox/rox/pkg/metrics"
	"go.etcd.io/bbolt"
)

var (
	bucketName = []byte("installationInfo")
)

// New creates a new store for BoltDB
func New(db *bbolt.DB) *store {
	return &store{underlying: singletonstore.New(db, bucketName, func() proto.Message {
		return new(storage.InstallationInfo)
	}, "InstallationInfo")}
}

type store struct {
	underlying singletonstore.SingletonStore
}

func (s *store) Get(_ context.Context) (*storage.InstallationInfo, bool, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "InstallationInfo")
	msg, err := s.underlying.Get()
	if err != nil {
		return nil, false, err
	}
	if msg == nil {
		return nil, false, nil
	}
	return msg.(*storage.InstallationInfo), true, nil
}

func (s *store) Upsert(_ context.Context, installationinfo *storage.InstallationInfo) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "InstallationInfo")
	return s.underlying.Create(installationinfo)
}
