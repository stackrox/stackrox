//go:build sql_integration

package datastore

import (
	"testing"

	clusterInitStore "github.com/stackrox/rox/central/clusterinit/store"
)

// For certain tests (central/sensor/service/service_impl_test.go) we need to interact
// with a cluster data store and the underlying cluster init store at the same time.
// For this purpose we need a way for these tests to extract the cluster init store
// from a cluster data store.
func IntrospectClusterInitStore(t *testing.T, storeInterface DataStore) clusterInitStore.Store {
	store, ok := storeInterface.(*datastoreImpl)
	if !ok {
		t.Fatal("unexpected datastore type")
	}
	return store.clusterInitStore
}
