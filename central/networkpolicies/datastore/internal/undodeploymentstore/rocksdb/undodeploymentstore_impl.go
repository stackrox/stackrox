package rocksdb

import (
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore"
)

var _ undodeploymentstore.UndoDeploymentStore = (*storeImpl)(nil)
