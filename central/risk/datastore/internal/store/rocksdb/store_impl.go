package rocksdb

import "github.com/stackrox/stackrox/central/risk/datastore/internal/store"

var _ store.Store = (*storeImpl)(nil)
