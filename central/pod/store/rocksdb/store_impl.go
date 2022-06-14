package rocksdb

import "github.com/stackrox/rox/central/pod/store"

var _ store.Store = (*storeImpl)(nil)
