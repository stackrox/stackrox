package test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/compliance"
	. "github.com/stackrox/rox/central/compliance/datastore"
	storeMocks "github.com/stackrox/rox/central/compliance/datastore/internal/store/mocks"
	"github.com/stackrox/rox/central/compliance/datastore/mocks"
	"github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

var (
	errFake = errors.New("fake error")
)

func TestComplianceDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(complianceDataStoreTestSuite))
}

type complianceDataStoreTestSuite struct {
	suite.Suite

	hasReadCtx  context.Context
	hasWriteCtx context.Context

	mockCtrl    *gomock.Controller
	mockFilter  *mocks.MockSacFilter
	mockStorage *storeMocks.MockStore

	dataStore DataStore
}

func (s *complianceDataStoreTestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))

	s.mockCtrl = gomock.NewController(s.T())
	s.mockFilter = mocks.NewMockSacFilter(s.mockCtrl)
	s.mockStorage = storeMocks.NewMockStore(s.mockCtrl)

	s.dataStore = NewDataStore(s.mockStorage, s.mockFilter)
}

func (s *complianceDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *complianceDataStoreTestSuite) TestGetLatestRunResults() {
	clusterID := "cid"
	standardID := "CIS_Docker_v1_2_0"
	expectedReturn := types.ResultsWithStatus{
		LastSuccessfulResults: &storage.ComplianceRunResults{},
	}

	// Expect storage fetch since filtering is performed afterwards.
	s.mockFilter.EXPECT().FilterRunResults(s.hasReadCtx, expectedReturn.LastSuccessfulResults).Return(expectedReturn.LastSuccessfulResults, nil)

	// Expect storage fetch.
	s.mockStorage.EXPECT().GetLatestRunResults(clusterID, standardID, types.WithMessageStrings).Return(expectedReturn, nil)

	// Call tested.
	result, err := s.dataStore.GetLatestRunResults(s.hasReadCtx, clusterID, standardID, types.WithMessageStrings)

	// Check results match.
	s.Nil(err)
	s.Equal(expectedReturn, result)
}

func (s *complianceDataStoreTestSuite) TestGetLatestRunResultsBatch() {
	clusterIDs := []string{"cid"}
	standardIDs := []string{"CIS_Docker_v1_2_0"}
	csPair := compliance.ClusterStandardPair{
		ClusterID:  "cid",
		StandardID: "CIS_Docker_v1_2_0",
	}
	expectedReturn := map[compliance.ClusterStandardPair]types.ResultsWithStatus{
		csPair: {
			LastSuccessfulResults: &storage.ComplianceRunResults{
				DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
					"dep1": {},
					"dep2": {},
					"dep3": {},
				},
			},
		},
	}

	// Expect storage fetch since filtering is performed afterwards.
	s.mockFilter.EXPECT().FilterBatchResults(s.hasReadCtx, expectedReturn).Return(expectedReturn, nil)

	// Expect storage fetch.
	s.mockStorage.EXPECT().GetLatestRunResultsBatch(clusterIDs, standardIDs, types.WithMessageStrings).Return(expectedReturn, nil)

	// Call tested.
	result, err := s.dataStore.GetLatestRunResultsBatch(s.hasReadCtx, clusterIDs, standardIDs, types.WithMessageStrings)

	// Check results match.
	s.Nil(err)
	s.Equal(1, len(result))
	s.Equal(expectedReturn[csPair], result[csPair])
}

func (s *complianceDataStoreTestSuite) TestGetLatestRunResultsFiltered() {
	csPair := compliance.ClusterStandardPair{
		ClusterID:  "cid",
		StandardID: "CIS_Docker_v1_2_0",
	}
	expectedReturn := map[compliance.ClusterStandardPair]types.ResultsWithStatus{
		csPair: {
			LastSuccessfulResults: &storage.ComplianceRunResults{
				DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
					"dep1": {},
					"dep2": {},
					"dep3": {},
				},
			},
		},
	}

	// Expect storage fetch since filtering is performed afterwards.
	s.mockFilter.EXPECT().FilterBatchResults(s.hasReadCtx, expectedReturn).Return(expectedReturn, nil)

	// Expect storage fetch.
	s.mockStorage.EXPECT().GetLatestRunResultsByClusterAndStandard(gomock.Any(), gomock.Any(), types.WithMessageStrings).Return(expectedReturn, nil)

	// Call tested.
	clusterIDs := []string{csPair.ClusterID}
	standardIDs := []string{csPair.StandardID}
	result, err := s.dataStore.GetLatestRunResultsForClustersAndStandards(s.hasReadCtx, clusterIDs, standardIDs, types.WithMessageStrings)

	// Check results match.
	s.Nil(err)
	s.Equal(1, len(result))
	s.Equal(expectedReturn[csPair], result[csPair])
}

func (s *complianceDataStoreTestSuite) TestStoreRunResults() {
	rr := &storage.ComplianceRunResults{}
	s.mockStorage.EXPECT().ClearAggregationResults()
	s.mockStorage.EXPECT().StoreRunResults(rr).Return(errFake)

	err := s.dataStore.StoreRunResults(s.hasWriteCtx, rr)

	s.Equal(errFake, err)
}

func (s *complianceDataStoreTestSuite) TestStoreFailure() {
	md := &storage.ComplianceRunMetadata{}
	s.mockStorage.EXPECT().StoreFailure(md).Return(errFake)

	err := s.dataStore.StoreFailure(s.hasWriteCtx, md)

	s.Equal(errFake, err)
}

func TestComplianceDataStoreWithSAC(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(complianceDataStoreWithSACTestSuite))
}

type complianceDataStoreWithSACTestSuite struct {
	suite.Suite

	hasReadCtx context.Context
	hasNoneCtx context.Context

	mockCtrl    *gomock.Controller
	mockFilter  *mocks.MockSacFilter
	mockStorage *storeMocks.MockStore

	dataStore DataStore
}

func (s *complianceDataStoreWithSACTestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())

	s.mockCtrl = gomock.NewController(s.T())
	s.mockFilter = mocks.NewMockSacFilter(s.mockCtrl)
	s.mockStorage = storeMocks.NewMockStore(s.mockCtrl)

	s.dataStore = NewDataStore(s.mockStorage, s.mockFilter)
}

func (s *complianceDataStoreWithSACTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *complianceDataStoreWithSACTestSuite) TestEnforceGetLatestRunResults() {
	// Expect no storage fetch.
	s.mockStorage.EXPECT().GetLatestRunResults(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// Call tested.
	clusterID := "cid"
	standardID := "CIS_Docker_v1_2_0"
	_, err := s.dataStore.GetLatestRunResults(s.hasNoneCtx, clusterID, standardID, types.WithMessageStrings)

	// Check results match.
	s.ErrorIs(err, errorhelpers.ErrNotFound)
}

func (s *complianceDataStoreWithSACTestSuite) TestEnforceStoreRunResults() {
	s.mockStorage.EXPECT().StoreRunResults(gomock.Any()).Times(0)

	err := s.dataStore.StoreRunResults(s.hasReadCtx, &storage.ComplianceRunResults{})

	s.ErrorIs(err, sac.ErrResourceAccessDenied)
}

func (s *complianceDataStoreWithSACTestSuite) TestEnforceStoreFailure() {
	s.mockStorage.EXPECT().StoreFailure(gomock.Any()).Times(0)

	err := s.dataStore.StoreFailure(s.hasReadCtx, &storage.ComplianceRunMetadata{})

	s.ErrorIs(err, sac.ErrResourceAccessDenied)
}

func (s *complianceDataStoreWithSACTestSuite) TestDoesNotUseStoredAggregationsWithSAC() {
	noop := func() ([]*storage.ComplianceAggregation_Result, []*storage.ComplianceAggregation_Source, map[*storage.ComplianceAggregation_Result]*storage.ComplianceDomain, error) {
		return nil, nil, nil, nil
	}
	aggArgs := &StoredAggregationArgs{
		QueryString:     "query",
		GroupBy:         nil,
		Unit:            storage.ComplianceAggregation_CLUSTER,
		AggregationFunc: noop,
	}
	_, _, _, err := s.dataStore.PerformStoredAggregation(context.Background(), aggArgs)
	s.Require().NoError(err)
}

func (s *complianceDataStoreWithSACTestSuite) TestUsesStoredAggregationsWithoutSAC() {
	s.T().Skip("ROX-9134: Re-enable or delete")

	queryString := "query"
	testUnit := storage.ComplianceAggregation_CLUSTER
	results := []*storage.ComplianceAggregation_Result{}
	sources := []*storage.ComplianceAggregation_Source{}
	domainMap := map[*storage.ComplianceAggregation_Result]*storage.ComplianceDomain{}
	s.mockStorage.EXPECT().GetAggregationResult(queryString, gomock.Nil(), testUnit).Return(results, sources, domainMap, nil)
	noop := func() ([]*storage.ComplianceAggregation_Result, []*storage.ComplianceAggregation_Source, map[*storage.ComplianceAggregation_Result]*storage.ComplianceDomain, error) {
		s.True(false, "The aggregation method should not be called when we find a stored result")
		return nil, nil, nil, nil
	}
	aggArgs := &StoredAggregationArgs{
		QueryString:     queryString,
		GroupBy:         nil,
		Unit:            testUnit,
		AggregationFunc: noop,
	}
	_, _, _, err := s.dataStore.PerformStoredAggregation(context.Background(), aggArgs)
	s.Require().NoError(err)
}
