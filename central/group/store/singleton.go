package store

import (
	"sync"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/generated/api/v1"
)

var (
	storage Store
	once    sync.Once
)

// Singleton returns the singleton group role mapper.
func Singleton() Store {
	once.Do(func() {
		storage = New(globaldb.GetGlobalDB())

		// Check to see that a global default exists.
		globalDefault, err := storage.Get(&v1.GroupProperties{})
		if err != nil {
			panic(err)
		}
		if globalDefault != nil {
			return
		}

		// If not, add admin as global default.
		err = storage.Upsert(&v1.Group{RoleName: role.Admin})
		if err != nil {
			panic(err)
		}
	})
	return storage
}
