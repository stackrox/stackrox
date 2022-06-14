package rocksdb

import "github.com/stackrox/rox/central/risk/datastore/internal/store"

var _ store.Store = (*storeImpl)(nil)
