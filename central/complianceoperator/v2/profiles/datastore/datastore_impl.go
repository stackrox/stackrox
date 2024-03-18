package datastore

import (
	"context"

	profileSearch "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/search"
	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

var (
	complianceSAC = sac.ForResource(resources.Compliance)
)

type datastoreImpl struct {
	db       pgPkg.DB
	store    pgStore.Store
	searcher profileSearch.Searcher
}

// GetProfile returns the profile for the given profile ID
func (d *datastoreImpl) GetProfile(ctx context.Context, profileID string) (*storage.ComplianceOperatorProfileV2, bool, error) {
	return d.store.Get(ctx, profileID)
}

// SearchProfiles returns the profiles for the given query
func (d *datastoreImpl) SearchProfiles(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorProfileV2, error) {
	return d.store.GetByQuery(ctx, query)
}

// UpsertProfile adds the profile to the database.  If enabling the use of this
// method from a service, the creation of the `ProfileRefID` must be accounted for.  In reality this
// method should only be used by the pipeline as this is a compliance operator object we are storing.
func (d *datastoreImpl) UpsertProfile(ctx context.Context, profile *storage.ComplianceOperatorProfileV2) error {
	if !complianceSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).IsAllowed(sac.ClusterScopeKey(profile.GetClusterId())) {
		return sac.ErrResourceAccessDenied
	}

	return d.store.Upsert(ctx, profile)
}

// DeleteProfileForCluster removes a profile from the database
func (d *datastoreImpl) DeleteProfileForCluster(ctx context.Context, uid string, clusterID string) error {
	if !complianceSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).IsAllowed(sac.ClusterScopeKey(clusterID)) {
		return sac.ErrResourceAccessDenied
	}

	_, err := d.store.DeleteByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).
		AddDocIDs(uid).ProtoQuery())
	return err
}

// DeleteProfilesByCluster deletes profiles of cluster with a specific id
func (d *datastoreImpl) DeleteProfilesByCluster(ctx context.Context, clusterID string) error {
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, clusterID).ProtoQuery()
	_, err := d.store.DeleteByQuery(ctx, query)
	if err != nil {
		return err
	}
	return nil
}

// GetProfilesByClusters gets the list of profiles for a given clusters
func (d *datastoreImpl) GetProfilesByClusters(ctx context.Context, clusterIDs []string) ([]*storage.ComplianceOperatorProfileV2, error) {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterIDs...).
		WithPagination(search.NewPagination().
			AddSortOption(search.NewSortOption(search.ComplianceOperatorProfileName))).
		ProtoQuery()

	return d.store.GetByQuery(ctx, query)
}

// CountProfiles returns count of profiles matching query
func (d *datastoreImpl) CountProfiles(ctx context.Context, q *v1.Query) (int, error) {
	return d.searcher.Count(ctx, q)
}
