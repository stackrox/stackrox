package service

import (
	"github.com/stackrox/rox/central/externalbackups/manager"
	backupStore "github.com/stackrox/rox/central/externalbackups/store"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(backupStore.Singleton(), initializeManager())
}

func initializeManager() manager.Manager {
	backups, err := backupStore.Singleton().ListBackups()
	if err != nil {
		panic(err)
	}
	mgr := manager.New()
	for _, b := range backups {
		if err := mgr.Upsert(b); err != nil {
			log.Errorf("error initializing backup: %v", err)
		}
	}
	return mgr
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
