package store

import (
	"sync"

	"github.com/stackrox/rox/central/globaldb"
	rolePkg "github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	store Store
	once  sync.Once

	log = logging.LoggerForModule()
)

// Singleton returns the singleton providing access to the roles store.
func Singleton() Store {
	once.Do(func() {
		store = fillPreloadedRoles(New(globaldb.GetGlobalDB()))
	})
	return store
}

// Adds the pre-defined roles to the store.
// Don't set default role here, that will cause us to overwrite the set default.
// We check that a default is set at runtime and return admin if not.
func fillPreloadedRoles(st Store) Store {
	inSto := st.(*storeImpl)
	// Load in all the preloaded roles
	for _, role := range rolePkg.DefaultRolesByName {
		if err := inSto.roleCrud.Upsert(role); err != nil {
			log.Panicf("Cannot upsert pre-defined role %s: %v", role.GetName(), err)
		}
	}
	return st
}
