// This file was originally generated with
// //go:generate cp ../../../../central/installation/store/bolt/store.go store_impl.go

package legacy

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper/singletonstore"
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
	return s.underlying.Create(installationinfo)
}
