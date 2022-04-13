package rocksdb

import (
	"github.com/stackrox/stackrox/central/networkpolicies/datastore/internal/undodeploymentstore"
)

var _ undodeploymentstore.UndoDeploymentStore = (*storeImpl)(nil)
