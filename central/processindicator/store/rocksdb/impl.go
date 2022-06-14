package rocksdb

import "github.com/stackrox/rox/central/processindicator/store"

var _ store.Store = (*storeImpl)(nil)
