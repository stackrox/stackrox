package service

import (
	"github.com/stackrox/rox/central/globaldb/v2backuprestore/manager"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	instance     Service
	instanceInit sync.Once
)

// Singleton returns the singleton instance of the v2 db backup/restore manager.
func Singleton() Service {
	instanceInit.Do(func() {
		instance = New(manager.Singleton())
	})
	return instance
}
