package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/search"
	edge "github.com/stackrox/rox/central/complianceoperator/v2/profiles/profileclusteredge/store/postgres"
	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// DataStore is the entry point for storing/retrieving compliance operator profiles.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// GetProfile returns the profile for the given profile ID
	GetProfile(ctx context.Context, profileID string) (*storage.ComplianceOperatorProfileV2, bool, error)

	// SearchProfiles returns the profiles for the given query
	SearchProfiles(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorProfileV2, error)

	// UpsertProfile adds the profile to the database
	UpsertProfile(ctx context.Context, result *storage.ComplianceOperatorProfileV2, clusterID string, profileUID string) error

	// DeleteProfileForCluster removes a profile from the database
	DeleteProfileForCluster(ctx context.Context, uid string, clusterID string) error

	// GetProfileEdgesByCluster gets the list of profile edges for a given cluster
	GetProfileEdgesByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorProfileClusterEdge, error)

	// CountProfiles returns count of profiles matching query
	CountProfiles(ctx context.Context, q *v1.Query) (int, error)
}

// New returns an instance of DataStore.
func New(complianceProfileStorage pgStore.Store, profileEdgeStore edge.Store, pool postgres.DB, searcher search.Searcher) DataStore {
	ds := &datastoreImpl{
		store:            complianceProfileStorage,
		profileEdgeStore: profileEdgeStore,
		db:               pool,
		searcher:         searcher,
	}
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB, searcher search.Searcher) (DataStore, error) {
	store := pgStore.New(pool)
	edgeStore := edge.New(pool)
	return New(store, edgeStore, pool, searcher), nil
}
