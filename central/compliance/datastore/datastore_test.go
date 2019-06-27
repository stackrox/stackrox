package datastore

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/compliance"
	storeMocks "github.com/stackrox/rox/central/compliance/datastore/internal/store/mocks"
	"github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
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

	dataStore DataStore
	storage   *storeMocks.MockStore

	mockCtrl *gomock.Controller
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
	s.storage = storeMocks.NewMockStore(s.mockCtrl)

	s.dataStore = &datastoreImpl{
		boltStore: s.storage,
	}
}

func (s *complianceDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *complianceDataStoreTestSuite) TestGetLatestRunResults() {
	// Expect storage fetch.
	clusterID := "cid"
	standardID := "sid"
	expectedReturn := types.ResultsWithStatus{
		LastSuccessfulResults: &storage.ComplianceRunResults{},
	}
	s.storage.EXPECT().GetLatestRunResults(clusterID, standardID, types.WithMessageStrings).Return(expectedReturn, nil)

	// Call tested.
	result, err := s.dataStore.GetLatestRunResults(s.hasReadCtx, clusterID, standardID, types.WithMessageStrings)

	// Check results match.
	s.Nil(err)
	s.Equal(fromInternalResultsWithStatus(expectedReturn), result)
}

func (s *complianceDataStoreTestSuite) TestGetLatestRunResultsBatch() {
	// Expect storage fetch.
	clusterIDs := []string{"cid"}
	standardIDs := []string{"sid"}
	csPair := compliance.ClusterStandardPair{
		ClusterID:  "cid",
		StandardID: "sid",
	}
	expectedReturn := map[compliance.ClusterStandardPair]types.ResultsWithStatus{
		csPair: {LastSuccessfulResults: &storage.ComplianceRunResults{}},
	}
	s.storage.EXPECT().GetLatestRunResultsBatch(clusterIDs, standardIDs, types.WithMessageStrings).Return(expectedReturn, nil)

	// Call tested.
	result, err := s.dataStore.GetLatestRunResultsBatch(s.hasReadCtx, clusterIDs, standardIDs, types.WithMessageStrings)

	// Check results match.
	s.Nil(err)
	s.Equal(1, len(result))
	s.Equal(fromInternalResultsWithStatus(expectedReturn[csPair]), result[csPair])
}

func (s *complianceDataStoreTestSuite) TestGetLatestRunResultsFiltered() {
	// Expect storage fetch.
	csPair := compliance.ClusterStandardPair{
		ClusterID:  "cid",
		StandardID: "sid",
	}
	expectedReturn := map[compliance.ClusterStandardPair]types.ResultsWithStatus{
		csPair: {LastSuccessfulResults: &storage.ComplianceRunResults{}},
	}
	s.storage.EXPECT().GetLatestRunResultsFiltered(gomock.Any(), gomock.Any(), types.WithMessageStrings).Return(expectedReturn, nil)

	// Call tested.
	clusterIDs := func(id string) bool { return true }
	standardIDs := func(id string) bool { return true }
	result, err := s.dataStore.GetLatestRunResultsFiltered(s.hasReadCtx, clusterIDs, standardIDs, types.WithMessageStrings)

	// Check results match.
	s.Nil(err)
	s.Equal(1, len(result))
	s.Equal(fromInternalResultsWithStatus(expectedReturn[csPair]), result[csPair])
}

func (s *complianceDataStoreTestSuite) TestStoreRunResults() {
	rr := &storage.ComplianceRunResults{}
	s.storage.EXPECT().StoreRunResults(rr).Return(errFake)

	err := s.dataStore.StoreRunResults(s.hasWriteCtx, rr)

	s.Equal(errFake, err)
}

func (s *complianceDataStoreTestSuite) TestStoreFailure() {
	md := &storage.ComplianceRunMetadata{}
	s.storage.EXPECT().StoreFailure(md).Return(errFake)

	err := s.dataStore.StoreFailure(s.hasWriteCtx, md)

	s.Equal(errFake, err)
}

func TestComplianceDataStoreWithSAC(t *testing.T) {
	t.Parallel()
	if !features.ScopedAccessControl.Enabled() {
		t.Skip()
	}
	suite.Run(t, new(complianceDataStoreWithSACTestSuite))
}

type complianceDataStoreWithSACTestSuite struct {
	suite.Suite

	hasReadCtx  context.Context
	hasWriteCtx context.Context
	hasNoneCtx  context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore

	mockCtrl *gomock.Controller
}

func (s *complianceDataStoreWithSACTestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Role)))
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)

	s.dataStore = &datastoreImpl{
		boltStore: s.storage,
	}
}

func (s *complianceDataStoreWithSACTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *complianceDataStoreWithSACTestSuite) TestEnforceGetLatestRunResults() {
	// Expect no storage fetch.
	s.storage.EXPECT().GetLatestRunResults(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// Call tested.
	clusterID := "cid"
	standardID := "sid"
	_, err := s.dataStore.GetLatestRunResults(s.hasNoneCtx, clusterID, standardID, types.WithMessageStrings)

	// Check results match.
	s.EqualError(err, "not found")
}

func (s *complianceDataStoreWithSACTestSuite) TestEnforceGetLatestRunResultsBatch() {
	// Expect storage fetch since filtering is performed afterwards.
	clusterIDs := []string{"cid"}
	standardIDs := []string{"sid"}
	csPair := compliance.ClusterStandardPair{
		ClusterID:  "cid",
		StandardID: "sid",
	}
	expectedReturn := map[compliance.ClusterStandardPair]types.ResultsWithStatus{
		csPair: {LastSuccessfulResults: &storage.ComplianceRunResults{}},
	}
	s.storage.EXPECT().GetLatestRunResultsBatch(clusterIDs, standardIDs, types.WithMessageStrings).Return(expectedReturn, nil)

	// Call tested.
	results, err := s.dataStore.GetLatestRunResultsBatch(s.hasNoneCtx, clusterIDs, standardIDs, types.WithMessageStrings)

	// Check results match.
	s.NoError(err)
	s.Empty(results)
}

func (s *complianceDataStoreWithSACTestSuite) TestEnforceGetLatestRunResultsFiltered() {
	// Expect storage fetch since filtering is performed afterwards.
	csPair := compliance.ClusterStandardPair{
		ClusterID:  "cid",
		StandardID: "sid",
	}
	expectedReturn := map[compliance.ClusterStandardPair]types.ResultsWithStatus{
		csPair: {LastSuccessfulResults: &storage.ComplianceRunResults{}},
	}
	s.storage.EXPECT().GetLatestRunResultsFiltered(gomock.Any(), gomock.Any(), types.WithMessageStrings).Return(expectedReturn, nil)

	// Call tested.
	clusterIDs := func(id string) bool { return true }
	standardIDs := func(id string) bool { return true }
	results, err := s.dataStore.GetLatestRunResultsFiltered(s.hasNoneCtx, clusterIDs, standardIDs, types.WithMessageStrings)

	// Check results match.
	s.NoError(err)
	s.Empty(results)
}

func (s *complianceDataStoreWithSACTestSuite) TestEnforceStoreRunResults() {
	s.storage.EXPECT().StoreRunResults(gomock.Any()).Times(0)

	rr := &storage.ComplianceRunResults{}
	err := s.dataStore.StoreRunResults(s.hasReadCtx, rr)

	s.EqualError(err, "permission denied")
}
