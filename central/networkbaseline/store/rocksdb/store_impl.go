package rocksdb

import (
	"github.com/stackrox/rox/central/networkbaseline/store"
)

var _ store.Store = (*storeImpl)(nil)
