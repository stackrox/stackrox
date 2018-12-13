package store

import (
	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	proto2 "github.com/stackrox/rox/pkg/bolthelper/crud/proto"
)

const rolesBucket = "roles"

// Store is the store for roles.
//go:generate mockgen-wrapper Store
type Store interface {
	GetRole(name string) (*storage.Role, error)
	GetRolesBatch(names []string) ([]*storage.Role, error)
	GetAllRoles() ([]*storage.Role, error)

	AddRole(*storage.Role) error
	UpdateRole(*storage.Role) error
	RemoveRole(name string) error
}

// New returns a new Store instance.
// The buckets used are not deduped, so separate instances will read and write to the same KV store.
func New(db *bbolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, rolesBucket)

	return &storeImpl{
		// Crud to store all of the roles.
		roleCrud: proto2.NewMessageCrud(db,
			rolesBucket,
			func(msg proto.Message) []byte { // Roles stored by name.
				return []byte(msg.(*storage.Role).GetName())
			},
			func() proto.Message {
				return &storage.Role{}
			},
		),
	}
}
