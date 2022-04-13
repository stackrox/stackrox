package rocksdb

import "github.com/stackrox/stackrox/central/pod/store"

var _ store.Store = (*storeImpl)(nil)
