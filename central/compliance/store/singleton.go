package store

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	storeInstance     Store
	storeInstanceInit sync.Once
)

// Singleton returns the compliance store singleton.
func Singleton() Store {
	storeInstanceInit.Do(func() {
		ds, err := NewBoltStore(globaldb.GetGlobalDB())
		utils.Must(err)
		storeInstance = ds
	})
	return storeInstance
}
