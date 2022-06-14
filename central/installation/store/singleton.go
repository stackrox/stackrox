package store

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/installation/store/bolt"
	"github.com/stackrox/stackrox/central/installation/store/postgres"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/uuid"
)

var (
	storeSingleton Store
	singletonInit  sync.Once
)

func createInitial() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
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
		if features.PostgresDatastore.Enabled() {
			storeSingleton = postgres.New(context.TODO(), globaldb.GetPostgres())
		} else {
			storeSingleton = bolt.New(globaldb.GetGlobalDB())
		}
		createInitial()
	})
	return storeSingleton
}
