package rocksdb

import (
	"github.com/stackrox/stackrox/central/networkbaseline/store"
)

var _ store.Store = (*storeImpl)(nil)
