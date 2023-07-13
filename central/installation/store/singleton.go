package store

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/installation/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	storeSingleton Store
	singletonInit  sync.Once
)

func createInitial() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.InstallationInfo)))

	_, exists, err := storeSingleton.Get(ctx)
	if err != nil {
		panic(err)
	}
	if exists {
		return
	}
	info := &storage.InstallationInfo{
		Id:      uuid.NewV4().String(),
		Created: types.TimestampNow(),
	}
	err = storeSingleton.Upsert(ctx, info)
	if err != nil {
		panic(err)
	}
}

// Singleton returns a singleton of the InstallationInfo store
func Singleton() Store {
	singletonInit.Do(func() {
		storeSingleton = pgStore.New(globaldb.GetPostgres())
		createInitial()
	})
	return storeSingleton
}
