package rocksdb

import "github.com/stackrox/stackrox/central/processindicator/store"

var _ store.Store = (*storeImpl)(nil)
