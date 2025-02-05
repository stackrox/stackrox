package store

import (
	"testing"

	clusterInitPostgres "github.com/stackrox/rox/central/clusterinit/store/postgres"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestClusterInitDataStore provides a datastore connected to postgres for testing purposes.
func GetTestClusterInitDataStore(_ *testing.T, pool postgres.DB) (Store, error) {
	return NewStore(clusterInitPostgres.New(pool)), nil
}
