package store

import (
	"sync"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/generated/storage"
)

var (
	as   Store
	once sync.Once
)

// Singleton returns the singleton group role mapper.
func Singleton() Store {
	once.Do(func() {
		as = New(globaldb.GetGlobalDB())

		// Check to see that a global default exists.
		globalDefault, err := as.Get(&storage.GroupProperties{})
		if err != nil {
			panic(err)
		}
		if globalDefault != nil {
			return
		}

		// If not, add admin as global default.
		err = as.Upsert(&storage.Group{RoleName: role.Admin})
		if err != nil {
			panic(err)
		}
	})
	return as
}
