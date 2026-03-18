package datastore

import (
	"context"
	"sort"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/env"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
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

// GetProfilesNames returns profile names that are present on every requested cluster, in
// tailored-first, non-tailored-second order (each bucket sorted A–Z). Pagination from q is
// applied after the Go-side merge.
//
// Fetch all profile objects for the accessible clusters in a single SQL query, then
// classify in Go. This avoids adding DB columns or extending the search framework to support
// COUNT(DISTINCT equivalence_hash) and operator_kind filtering. The data volume is bounded
// (O(profiles_per_cluster × clusters)) and acceptable for the profile picker use case.
func (d *datastoreImpl) GetProfilesNames(ctx context.Context, q *v1.Query, clusterIDs []string) ([]string, error) {
	readableClusterIDs := bestEffortClusters(ctx, clusterIDs)
	if len(readableClusterIDs) == 0 {
		return nil, nil
	}

	sacQ, err := withSACFilter(ctx, resources.Compliance, q)
	if err != nil {
		return nil, err
	}

	// Fetch full profile objects: cluster filter + SAC + user filter (no pagination — we paginate in Go).
	fetchQ := search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ClusterID, readableClusterIDs...).ProtoQuery(),
		withoutPagination(sacQ),
	)
	profiles, err := d.store.GetByQuery(ctx, fetchQ)
	if err != nil {
		return nil, err
	}

	skipHash := env.SkipTailoredProfileEquivalenceHash.BooleanSetting()
	tailoredNames, standardNames := resolveEligibleProfileNames(profiles, len(readableClusterIDs), skipHash)

	sort.Strings(tailoredNames)
	sort.Strings(standardNames)
	merged := append(tailoredNames, standardNames...)
	return applyPagination(merged, q.GetPagination()), nil
}

// CountDistinctProfiles returns the total number of distinct profile names that would be
// returned by GetProfilesNames for the same arguments (before pagination). Both functions
// must apply identical classification logic to keep the count consistent with the list.
func (d *datastoreImpl) CountDistinctProfiles(ctx context.Context, q *v1.Query, clusterIDs []string) (int, error) {
	readableClusterIDs := bestEffortClusters(ctx, clusterIDs)
	if len(readableClusterIDs) == 0 {
		return 0, nil
	}

	sacQ, err := withSACFilter(ctx, resources.Compliance, q)
	if err != nil {
		return 0, err
	}

	fetchQ := search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ClusterID, readableClusterIDs...).ProtoQuery(),
		withoutPagination(sacQ),
	)
	profiles, err := d.store.GetByQuery(ctx, fetchQ)
	if err != nil {
		return 0, err
	}

	skipHash := env.SkipTailoredProfileEquivalenceHash.BooleanSetting()
	tailoredNames, standardNames := resolveEligibleProfileNames(profiles, len(readableClusterIDs), skipHash)
	return len(tailoredNames) + len(standardNames), nil
}

// resolveEligibleProfileNames returns two disjoint name lists from the given profiles:
//   - tailoredNames: tailored profiles present on all n clusters with a consistent equivalence_hash.
//   - standardNames: non-tailored profiles present on all n clusters.
//
// Names with mixed kinds (tailored on one cluster, non-tailored on another) are excluded
// from both lists. Hash equivalence: all-empty hash is treated as equivalent (sensor
// fallback). The bypass env var disables hash checking entirely.
func resolveEligibleProfileNames(
	profiles []*storage.ComplianceOperatorProfileV2,
	n int,
	skipHash bool,
) (tailoredNames, standardNames []string) {
	byName := groupProfilesByName(profiles)
	universal := retainPresentOnAllClusters(byName, n)
	tailoredGroups, standardGroups := partitionByKind(universal)

	tailoredNames = tailoredNamesWithConsistentHash(tailoredGroups, skipHash)
	for name := range standardGroups {
		standardNames = append(standardNames, name)
	}
	return
}

// groupProfilesByName groups profiles into a map keyed by profile name.
func groupProfilesByName(profiles []*storage.ComplianceOperatorProfileV2) map[string][]*storage.ComplianceOperatorProfileV2 {
	byName := make(map[string][]*storage.ComplianceOperatorProfileV2, len(profiles))
	for _, p := range profiles {
		byName[p.GetName()] = append(byName[p.GetName()], p)
	}
	return byName
}

// retainPresentOnAllClusters keeps only name groups that have exactly n instances,
// meaning the profile exists on every one of the n requested clusters.
func retainPresentOnAllClusters(byName map[string][]*storage.ComplianceOperatorProfileV2, n int) map[string][]*storage.ComplianceOperatorProfileV2 {
	result := make(map[string][]*storage.ComplianceOperatorProfileV2)
	for name, instances := range byName {
		if len(instances) == n {
			result[name] = instances
		}
	}
	return result
}

// partitionByKind splits groups into tailored-profile and non-tailored buckets.
// Groups with mixed kinds (tailored on one cluster, non-tailored on another) are excluded from both.
func partitionByKind(groups map[string][]*storage.ComplianceOperatorProfileV2) (tailoredGroups, standardGroups map[string][]*storage.ComplianceOperatorProfileV2) {
	tailoredGroups = make(map[string][]*storage.ComplianceOperatorProfileV2)
	standardGroups = make(map[string][]*storage.ComplianceOperatorProfileV2)
	for name, instances := range groups {
		allTP, allOOB := true, true
		for _, inst := range instances {
			if inst.GetOperatorKind() == storage.ComplianceOperatorProfileV2_TAILORED_PROFILE {
				allOOB = false
			} else {
				allTP = false
			}
		}
		switch {
		case allTP:
			tailoredGroups[name] = instances
		case allOOB:
			standardGroups[name] = instances
		}
	}
	return
}

// tailoredNamesWithConsistentHash returns the names of tailored profile groups where all
// instances share the same equivalence_hash. When skipHash is true, all names are returned.
func tailoredNamesWithConsistentHash(tailoredGroups map[string][]*storage.ComplianceOperatorProfileV2, skipHash bool) []string {
	var names []string
	for name, instances := range tailoredGroups {
		if skipHash || hashesEquivalent(instances) {
			names = append(names, name)
		}
	}
	return names
}

// hashesEquivalent returns true when all instances share the same equivalence_hash value.
// An all-empty hash is treated as equivalent (COUNT(DISTINCT "") = 1).
func hashesEquivalent(instances []*storage.ComplianceOperatorProfileV2) bool {
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

// withoutPagination returns a clone of q with pagination stripped.
// The caller applies pagination in Go after merging the tailored and non-tailored buckets.
func withoutPagination(q *v1.Query) *v1.Query {
	if q == nil {
		return nil
	}
	cloned := q.CloneVT()
	cloned.Pagination = nil
	return cloned
}

// applyPagination slices merged according to the pagination settings in p.
// Returns the full slice when p is nil or limit is 0.
func applyPagination(merged []string, p *v1.QueryPagination) []string {
	if p == nil {
		return merged
	}
	offset := int(p.GetOffset())
	limit := int(p.GetLimit())
	if offset >= len(merged) {
		return nil
	}
	merged = merged[offset:]
	if limit > 0 && limit < len(merged) {
		merged = merged[:limit]
	}
	return merged
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
