package datastore

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/group/datastore/internal/store/bolt"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	ds   DataStore
	once sync.Once
	log  = logging.LoggerForModule()
)

func initialize() {
	db := bolt.New(globaldb.GetGlobalDB())
	ds = New(db)

	// Give datastore access to groups so that it can delete any groups with empty props on startup
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))

	utils.Should(ds.RemoveAllWithEmptyProperties(ctx))

	// Create groups with empty properties.
	err := db.UpsertInvalidEntry(ctx, &storage.Group{
		Props:    nil,
		RoleName: "",
	})
	if err != nil {
		log.Errorf("Failed to upsert invalid group entry: %v", err)
	}
}

// Singleton returns the singleton providing access to the roles store.
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
