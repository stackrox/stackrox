package store

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	as   Store
	once sync.Once
)

// Singleton returns the singleton group role mapper.
func Singleton() Store {
	once.Do(func() {
		as = New(globaldb.GetGlobalDB())
	})
	return as
}
