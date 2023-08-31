package backend

import (
	"testing"

	"github.com/stackrox/rox/central/clusterinit/backend/certificate"
	"github.com/stackrox/rox/central/clusterinit/store"
	postgresStore "github.com/stackrox/rox/central/clusterinit/store/postgres"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresBackend provides a backend connected to postgres for testing purposes.
func GetTestPostgresBackend(_ *testing.T, pool postgres.DB) (Backend, error) {
	backendStore := store.NewStore(postgresStore.New(pool))
	certProvider := certificate.NewProvider()

	return newBackend(backendStore, certProvider), nil
}
