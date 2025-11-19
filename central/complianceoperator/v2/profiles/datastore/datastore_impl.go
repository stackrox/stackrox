package datastore

import (
	"context"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
)

var (
	complianceSAC = sac.ForResource(resources.Compliance)
)

type datastoreImpl struct {
	db    pgPkg.DB
	store pgStore.Store
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

	return d.store.DeleteByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).
		AddDocIDs(uid).ProtoQuery())
}

// DeleteProfilesByCluster deletes profiles of cluster with a specific id
func (d *datastoreImpl) DeleteProfilesByCluster(ctx context.Context, clusterID string) error {
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, clusterID).ProtoQuery()
	return d.store.DeleteByQuery(ctx, query)
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
	return d.store.Count(ctx, q)
}

type distinctProfileName struct {
	ProfileName string `db:"compliance_profile_name"`
}

// GetProfilesNames gets the list of distinct profile names for the query
func (d *datastoreImpl) GetProfilesNames(ctx context.Context, q *v1.Query, clusterIDs []string) ([]string, error) {
	// Build the matching query to restrict profiles to the incoming clusters
	readableClusterIDs := bestEffortClusters(ctx, clusterIDs)

	var err error
	q, err = withSACFilter(ctx, resources.Compliance, q)
	if err != nil {
		return nil, err
	}

	// We only want to return profiles that exist in EACH cluster requested.  This covers instances where a
	// profile may not exist on every cluster and thus we do not want to return it here.  So the
	// `AddNumericField` select is to ensure we get profiles that exist in each cluster passed in.  What
	// this results in is a having clause to ensure the count matches the number of clusters passed in.
	parsedQuery := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, readableClusterIDs...).
		AddNumericField(search.ProfileCount, storage.Comparator_EQUALS, float32(len(readableClusterIDs))).
		ProtoQuery()

	parsedQuery = search.ConjunctionQuery(
		parsedQuery,
		q,
	)

	// Build the select and group by on distinct profile name
	parsedQuery.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.ComplianceOperatorProfileName).Distinct().Proto(),
	}
	parsedQuery.GroupBy = &v1.QueryGroupBy{
		Fields: []string{
			search.ComplianceOperatorProfileName.String(),
		},
	}
	parsedQuery.Pagination = q.GetPagination()

	var results []*distinctProfileName
	results, err = pgSearch.RunSelectRequestForSchema[distinctProfileName](ctx, d.db, schema.ComplianceOperatorProfileV2Schema, parsedQuery)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	profileNames := make([]string, 0, len(results))
	for _, result := range results {
		profileNames = append(profileNames, result.ProfileName)
	}

	return profileNames, err
}

type distinctProfileCount struct {
	TotalCount int    `db:"compliance_profile_name_count"`
	Name       string `db:"compliance_profile_name"`
}

// CountDistinctProfiles returns count of distinct profiles matching query
func (d *datastoreImpl) CountDistinctProfiles(ctx context.Context, q *v1.Query, clusterIDs []string) (int, error) {
	// Build the matching query to restrict profiles to the incoming clusters
	readableClusterIDs := bestEffortClusters(ctx, clusterIDs)

	query := search.ConjunctionQuery(
		search.NewQueryBuilder().
			AddExactMatches(search.ClusterID, readableClusterIDs...).
			AddNumericField(search.ProfileCount, storage.Comparator_EQUALS, float32(len(readableClusterIDs))).ProtoQuery(),
		q,
	)

	query.GroupBy = &v1.QueryGroupBy{
		Fields: []string{
			search.ComplianceOperatorProfileName.String(),
		},
	}

	var results []*distinctProfileCount
	results, err := pgSearch.RunSelectRequestForSchema[distinctProfileCount](ctx, d.db, schema.ComplianceOperatorProfileV2Schema, withCountQuery(query, search.ComplianceOperatorProfileName))
	if err != nil {
		return 0, err
	}
	return len(results), nil
}

func withCountQuery(query *v1.Query, field search.FieldLabel) *v1.Query {
	cloned := query.CloneVT()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(field).AggrFunc(aggregatefunc.Count).Proto(),
	}
	return cloned
}

func bestEffortClusters(ctx context.Context, clusterIDs []string) []string {
	// Best effort SAC.  We only want to return profiles from the cluster list that the user has access to
	// view.  So we perform an access check to create a narrowed list instead of embedding it in the query,
	// as the logic is much simpler.
	var bestEffortClusters []string
	for _, clusterID := range clusterIDs {
		if complianceSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).IsAllowed(sac.ClusterScopeKey(clusterID)) {
			bestEffortClusters = append(bestEffortClusters, clusterID)
		}
	}

	return bestEffortClusters
}

func withSACFilter(ctx context.Context, targetResource permissions.ResourceMetadata, query *v1.Query) (*v1.Query, error) {
	sacQueryFilter, err := pgSearch.GetReadSACQuery(ctx, targetResource)
	if err != nil {
		return nil, err
	}
	return search.FilterQueryByQuery(query, sacQueryFilter), nil
}
