package store

import (
	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	protoCrud "github.com/stackrox/rox/pkg/bolthelper/crud/proto"
)

var rolesBucket = []byte("roles")

// Store is the store for roles.
//go:generate mockgen-wrapper
type Store interface {
	GetRole(name string) (*storage.Role, error)
	GetAllRoles() ([]*storage.Role, error)

	AddRole(*storage.Role) error
	UpdateRole(*storage.Role) error
	RemoveRole(name string) error
}

// New returns a new Store instance.
// The buckets used are not deduped, so separate instances will read and write to the same KV store.
func New(db *bbolt.DB) Store {
	return &storeImpl{
		// Crud to store all of the roles.
		roleCrud: protoCrud.NewMessageCrudOrPanic(db,
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
