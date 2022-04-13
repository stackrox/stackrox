package store

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/uuid"
)

var (
	storeSingleton Store
	singletonInit  sync.Once
	log            = logging.LoggerForModule()
)

// Singleton returns a singleton of the InstallationInfo store
func Singleton() Store {
	singletonInit.Do(func() {
		store := New(globaldb.GetGlobalDB())
		info, err := store.GetInstallationInfo()
		if err != nil {
			panic(err)
		}
		if info != nil {
			storeSingleton = store
			return
		}

		info = &storage.InstallationInfo{
			Id:      uuid.NewV4().String(),
			Created: types.TimestampNow(),
		}
		err = store.AddInstallationInfo(info)
		if err != nil {
			panic(err)
		}
		storeSingleton = store
	})
	return storeSingleton
}
