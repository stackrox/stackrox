package store

import (
	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	proto2 "github.com/stackrox/rox/pkg/bolthelper/crud/proto"
)

const rolesBucket = "roles"

// Store is the store for roles.
//go:generate mockgen-wrapper Store
type Store interface {
	GetRole(name string) (*v1.Role, error)
	GetRolesBatch(names []string) ([]*v1.Role, error)
	GetAllRoles() ([]*v1.Role, error)

	AddRole(*v1.Role) error
	UpdateRole(*v1.Role) error
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
				return []byte(msg.(*v1.Role).GetName())
			},
			func() proto.Message {
				return &v1.Role{}
			},
		),
	}
}
