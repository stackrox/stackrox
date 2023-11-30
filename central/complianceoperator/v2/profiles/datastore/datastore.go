package datastore

import (
	"context"
	"testing"

	edge "github.com/stackrox/rox/central/complianceoperator/v2/profiles/profileclusteredge/store/postgres"
	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for storing/retrieving compliance operator profiles.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// UpsertProfile adds the profile to the database
	UpsertProfile(ctx context.Context, result *storage.ComplianceOperatorProfileV2, clusterID string, profileUID string) error

	// DeleteProfileForCluster removes a profile from the database
	DeleteProfileForCluster(ctx context.Context, uid string, clusterID string) error

	// GetProfileEdgesByCluster gets the list of profile edges for a given cluster
	GetProfileEdgesByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorProfileClusterEdge, error)
}

// New returns an instance of DataStore.
func New(complianceProfileStorage pgStore.Store, profileEdgeStore edge.Store, pool postgres.DB) DataStore {
	ds := &datastoreImpl{
		store:            complianceProfileStorage,
		profileEdgeStore: profileEdgeStore,
		db:               pool,
	}
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) (DataStore, error) {
	store := pgStore.New(pool)
	edgeStore := edge.New(pool)
	return New(store, edgeStore, pool), nil
}
