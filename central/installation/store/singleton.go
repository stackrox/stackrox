package store

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
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

		// TODO: remove
		log.Infof("Installation info added with id: %s", info.Id)
		storeSingleton = store
	})
	return storeSingleton
}
