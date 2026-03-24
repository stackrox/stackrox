package datastore

import (
	"context"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
)

var (
	log           = logging.LoggerForModule()
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

	if !env.SkipTailoredProfileEquivalenceHash.BooleanSetting() {
		profileNames, err = d.filterNonEquivalentTPs(ctx, profileNames, readableClusterIDs)
		if err != nil {
			return nil, err
		}
	}

	return profileNames, nil
}

// filterNonEquivalentTPs removes tailored profile names whose instances have differing
// equivalence hashes across clusters. OOB profiles are passed through unchanged.
func (d *datastoreImpl) filterNonEquivalentTPs(ctx context.Context, names []string, clusterIDs []string) ([]string, error) {
	if len(names) == 0 {
		return names, nil
	}

	profiles, err := d.store.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterIDs...).
		AddExactMatches(search.ComplianceOperatorProfileName, names...).ProtoQuery())
	if err != nil {
		return nil, err
	}

	byName := make(map[string][]*storage.ComplianceOperatorProfileV2, len(names))
	for _, p := range profiles {
		byName[p.GetName()] = append(byName[p.GetName()], p)
	}

	return applyEquivalenceFilter(names, byName), nil
}

// applyEquivalenceFilter filters names using pre-fetched profile instances grouped by name.
// Tailored profiles whose instances have inconsistent equivalence hashes are removed.
// OOB profiles are passed through unchanged.
func applyEquivalenceFilter(names []string, byName map[string][]*storage.ComplianceOperatorProfileV2) []string {
	filtered := names[:0]
	for _, name := range names {
		instances := byName[name]
		isTP := len(instances) > 0 && instances[0].GetOperatorKind() == storage.ComplianceOperatorProfileV2_TAILORED_PROFILE
		if !isTP || TailoredProfilesEquivalent(instances) {
			filtered = append(filtered, name)
		} else {
			log.Warnf("Tailored profile %q excluded from profile picker: content differs across clusters (equivalence hash mismatch). "+
				"Deploy an identical tailored profile on all clusters to make it schedulable.", name)
		}
	}
	return filtered
}

type distinctProfileCount struct {
	TotalCount int    `db:"compliance_profile_name_count"`
	Name       string `db:"compliance_profile_name"`
}

// CountDistinctProfiles returns the number of distinct profile names present on all requested clusters.
func (d *datastoreImpl) CountDistinctProfiles(ctx context.Context, q *v1.Query, clusterIDs []string) (int, error) {
	readableClusterIDs := bestEffortClusters(ctx, clusterIDs)

	var err error
	q, err = withSACFilter(ctx, resources.Compliance, q)
	if err != nil {
		return 0, err
	}

	query := search.ConjunctionQuery(
		search.NewQueryBuilder().
			AddExactMatches(search.ClusterID, readableClusterIDs...).
			AddNumericField(search.ProfileCount, storage.Comparator_EQUALS, float32(len(readableClusterIDs))).
			ProtoQuery(),
		q,
	)
	query.GroupBy = &v1.QueryGroupBy{
		Fields: []string{
			search.ComplianceOperatorProfileName.String(),
		},
	}

	var results []*distinctProfileCount
	results, err = pgSearch.RunSelectRequestForSchema[distinctProfileCount](ctx, d.db, schema.ComplianceOperatorProfileV2Schema, withCountQuery(query, search.ComplianceOperatorProfileName))
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

// TailoredProfilesEquivalent returns true when all instances share the same equivalence_hash
// value. An all-empty hash is treated as equivalent (COUNT(DISTINCT "") = 1). An empty or nil
// slice is considered equivalent.
func TailoredProfilesEquivalent(instances []*storage.ComplianceOperatorProfileV2) bool {
	if len(instances) == 0 {
		return true
	}
	h := instances[0].GetEquivalenceHash()
	for _, inst := range instances[1:] {
		if inst.GetEquivalenceHash() != h {
			return false
		}
	}
	return true
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
