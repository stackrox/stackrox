package store

import (
	bolt "github.com/etcd-io/bbolt"
	gogoProto "github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/bolthelper/crud/proto"
)

var processWhitelistResultsBucket = []byte("processWhitelistResults")

// Store provides storage functionality for process whitelists
type Store interface {
	UpsertWhitelistResults(*storage.ProcessWhitelistResults) error
	GetWhitelistResults(deploymentID string) (*storage.ProcessWhitelistResults, error)
	DeleteWhitelistResults(deploymentID string) error
}

// New Returns a new instance of Store using a bolt DB
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, processWhitelistResultsBucket)
	return &store{
		crud: proto.NewMessageCrud(db, processWhitelistResultsBucket,
			func(msg gogoProto.Message) []byte {
				return []byte(msg.(*storage.ProcessWhitelistResults).GetDeploymentId())
			},
			func() gogoProto.Message {
				return new(storage.ProcessWhitelistResults)
			},
		),
	}
}
