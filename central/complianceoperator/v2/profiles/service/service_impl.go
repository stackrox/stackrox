package service

import (
	"context"
	"slices"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/benchmark"
	profileDS "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	"github.com/stackrox/rox/central/convert/storagetov2"
	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

const (
	maxPaginationLimit = 1000
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Compliance), permissions.View(resources.Cluster)): {
			v2.ComplianceProfileService_GetComplianceProfile_FullMethodName,
			v2.ComplianceProfileService_ListComplianceProfiles_FullMethodName,
			v2.ComplianceProfileService_ListProfileSummaries_FullMethodName,
		},
	})
)

// New returns a service object for registering with grpc.
func New(complianceProfilesDS profileDS.DataStore) Service {
	return &serviceImpl{
		complianceProfilesDS: complianceProfilesDS,
	}
}

type serviceImpl struct {
	v2.UnimplementedComplianceProfileServiceServer

	complianceProfilesDS profileDS.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v2.RegisterComplianceProfileServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterComplianceProfileServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetComplianceProfile retrieves the specified compliance profile
func (s *serviceImpl) GetComplianceProfile(ctx context.Context, req *v2.ResourceByID) (*v2.ComplianceProfile, error) {
	if req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Scan configuration name is required for retrieval")
	}

	profile, found, err := s.complianceProfilesDS.GetProfile(ctx, req.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve compliance profile with id %q.", req.GetId())
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "compliance profile with id %q does not exist", req.GetId())
	}

	// Get the benchmark
	profileBenchmark, err := benchmark.GetBenchmarkFromProfile(profile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve benchmarks for profile %q.", profile.GetName())
	}

	return storagetov2.ComplianceV2Profile(profile, []*storage.ComplianceOperatorBenchmarkV2{profileBenchmark}), nil
}

// ListComplianceProfiles returns profiles matching given query
func (s *serviceImpl) ListComplianceProfiles(ctx context.Context, request *v2.ProfilesForClusterRequest) (*v2.ListComplianceProfilesResponse, error) {
	if request.GetClusterId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "cluster is required")
	}

	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// Add the cluster ids as an exact match
	parsedQuery = search.ConjunctionQuery(
		search.NewQueryBuilder().AddSelectFields().AddExactMatches(search.ClusterID, request.GetClusterId()).ProtoQuery(),
		parsedQuery,
	)

	// To get total count, need the parsed query without the paging.
	countQuery := parsedQuery.CloneVT()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, request.GetQuery().GetPagination(), maxPaginationLimit)

	profiles, err := s.complianceProfilesDS.SearchProfiles(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance profiles for cluster %v", request.GetClusterId())
	}

	// Get the benchmarks
	benchmarkMap, err := s.getBenchmarks(ctx, profiles)
	if err != nil {
		return nil, err
	}

	totalCount, err := s.complianceProfilesDS.CountProfiles(ctx, countQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance profiles counts for %v", request)
	}

	return &v2.ListComplianceProfilesResponse{
		Profiles:   storagetov2.ComplianceV2Profiles(profiles, benchmarkMap),
		TotalCount: int32(totalCount),
	}, nil
}

// ListProfileSummaries returns profile summaries matching incoming clusters
func (s *serviceImpl) ListProfileSummaries(ctx context.Context, request *v2.ClustersProfileSummaryRequest) (*v2.ListComplianceProfileSummaryResponse, error) {
	if len(request.GetClusterIds()) == 0 {
		return nil, errors.Wrap(errox.InvalidArgs, "cluster is required")
	}

	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to parse query %v", err)
	}

	// To get total count, need the parsed query without the paging.
	countQuery := parsedQuery.CloneVT()

	// Fill in pagination.
	paginated.FillPaginationV2(parsedQuery, request.GetQuery().GetPagination(), maxPaginationLimit)
	// make sure we sort by profile name at a minimum
	if parsedQuery.GetPagination().GetSortOptions() == nil {
		parsedQuery.Pagination.SortOptions = []*v1.QuerySortOption{
			{
				Field: search.ComplianceOperatorProfileName.String(),
			},
		}
	}

	profileNames, err := s.complianceProfilesDS.GetProfilesNames(ctx, parsedQuery, request.GetClusterIds())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance profiles for %v", request)
	}

	// Build query to get the filtered list by profile names
	profileQuery := search.NewQueryBuilder().AddSelectFields().AddExactMatches(search.ComplianceOperatorProfileName, profileNames...).ProtoQuery()
	// Bring the sort options only, paging is handled in step one when we get the distinct profiles.
	profileQuery.Pagination = &v1.QueryPagination{}
	profileQuery.Pagination.SortOptions = parsedQuery.GetPagination().GetSortOptions()

	profiles, err := s.complianceProfilesDS.SearchProfiles(ctx, profileQuery)
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance profiles for %v", request)
	}

	// Filter out TailoredProfiles that have different configurations across clusters.
	// For OOB profiles, name match is sufficient. For TailoredProfiles, we check
	// that tailored_details are equivalent across all clusters.
	profiles = filterEquivalentProfiles(profiles, len(request.GetClusterIds()))

	// Get the benchmarks
	benchmarkMap, err := s.getBenchmarks(ctx, profiles)
	if err != nil {
		return nil, err
	}

	totalCount, err := s.complianceProfilesDS.CountDistinctProfiles(ctx, countQuery, request.GetClusterIds())
	if err != nil {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unable to retrieve compliance profiles counts for %v", request)
	}

	return &v2.ListComplianceProfileSummaryResponse{
		Profiles:   storagetov2.ComplianceProfileSummary(profiles, benchmarkMap),
		TotalCount: int32(totalCount),
	}, nil
}

func (s *serviceImpl) getBenchmarks(ctx context.Context, profiles []*storage.ComplianceOperatorProfileV2) (map[string][]*storage.ComplianceOperatorBenchmarkV2, error) {
	// Get the benchmarks
	benchmarkMap := make(map[string][]*storage.ComplianceOperatorBenchmarkV2, len(profiles))
	for _, profile := range profiles {
		if _, found := benchmarkMap[profile.GetName()]; !found {
			profileBenchmark, err := benchmark.GetBenchmarkFromProfile(profile)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to retrieve benchmarks for profile %q.", profile.GetName())
			}
			benchmarkMap[profile.GetName()] = []*storage.ComplianceOperatorBenchmarkV2{profileBenchmark}
		}
	}

	return benchmarkMap, nil
}

// filterEquivalentProfiles filters out profiles that are not equivalent across clusters.
// For OOB profiles, name equality is sufficient (already handled by GetProfilesNames).
// For TailoredProfiles, we need to check that tailored_details are equivalent across all clusters.
func filterEquivalentProfiles(profiles []*storage.ComplianceOperatorProfileV2, numClusters int) []*storage.ComplianceOperatorProfileV2 {
	if numClusters <= 1 {
		return profiles
	}

	// Group profiles by name
	profilesByName := make(map[string][]*storage.ComplianceOperatorProfileV2)
	for _, p := range profiles {
		profilesByName[p.GetName()] = append(profilesByName[p.GetName()], p)
	}

	var result []*storage.ComplianceOperatorProfileV2
	for name, group := range profilesByName {
		// Skip if not present in all clusters
		if len(group) < numClusters {
			continue
		}

		// Check if any profile in the group is a TailoredProfile
		hasTailoredProfile := false
		for _, p := range group {
			if p.GetTailoredDetails() != nil {
				hasTailoredProfile = true
				break
			}
		}

		if !hasTailoredProfile {
			// OOB profiles - name match is sufficient, keep first one for deduplication
			result = append(result, group[0])
		} else {
			// TailoredProfiles - check if all instances are equivalent
			if areTailoredProfilesEquivalent(group) {
				result = append(result, group[0])
			}
			// If not equivalent, don't include this profile (silently filtered out)
			_ = name // suppress unused warning
		}
	}

	// Sort by name to maintain consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].GetName() < result[j].GetName()
	})

	return result
}

// areTailoredProfilesEquivalent checks if all TailoredProfiles in the group have equivalent configurations.
// Two TailoredProfiles are considered equivalent if they have:
// - Same base profile (extends)
// - Same disabled rules (by name)
// - Same enabled rules (by name)
// - Same manual rules (by name)
// - Same set values (by name and value)
func areTailoredProfilesEquivalent(profiles []*storage.ComplianceOperatorProfileV2) bool {
	if len(profiles) <= 1 {
		return true
	}

	// Use first profile as reference
	ref := profiles[0].GetTailoredDetails()

	for _, p := range profiles[1:] {
		td := p.GetTailoredDetails()

		// Both must be TailoredProfiles (or both OOB)
		if (ref == nil) != (td == nil) {
			return false
		}

		// If both are OOB (nil tailored_details), they're equivalent by name
		if ref == nil && td == nil {
			continue
		}

		// Check base profile
		if ref.GetExtends() != td.GetExtends() {
			return false
		}

		// Check disabled rules
		if !areRuleModificationsEquivalent(ref.GetDisabledRules(), td.GetDisabledRules()) {
			return false
		}

		// Check enabled rules
		if !areRuleModificationsEquivalent(ref.GetEnabledRules(), td.GetEnabledRules()) {
			return false
		}

		// Check manual rules
		if !areRuleModificationsEquivalent(ref.GetManualRules(), td.GetManualRules()) {
			return false
		}

		// Check set values
		if !areValueOverridesEquivalent(ref.GetSetValues(), td.GetSetValues()) {
			return false
		}
	}

	return true
}

// areRuleModificationsEquivalent checks if two slices of rule modifications are equivalent.
// Rules are compared by name only (rationale may differ across clusters).
func areRuleModificationsEquivalent(a, b []*storage.StorageTailoredProfileRuleModification) bool {
	if len(a) != len(b) {
		return false
	}

	// Extract and sort names
	namesA := make([]string, len(a))
	namesB := make([]string, len(b))
	for i, r := range a {
		namesA[i] = r.GetName()
	}
	for i, r := range b {
		namesB[i] = r.GetName()
	}
	slices.Sort(namesA)
	slices.Sort(namesB)

	return slices.Equal(namesA, namesB)
}

// areValueOverridesEquivalent checks if two slices of value overrides are equivalent.
// Values are compared by name and value (rationale may differ).
func areValueOverridesEquivalent(a, b []*storage.StorageTailoredProfileValueOverride) bool {
	if len(a) != len(b) {
		return false
	}

	// Create maps for comparison
	mapA := make(map[string]string, len(a))
	for _, v := range a {
		mapA[v.GetName()] = v.GetValue()
	}

	for _, v := range b {
		if val, exists := mapA[v.GetName()]; !exists || val != v.GetValue() {
			return false
		}
	}

	return true
}
