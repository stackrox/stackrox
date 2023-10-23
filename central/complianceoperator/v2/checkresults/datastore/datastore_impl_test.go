//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	checkResultsStorage "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestComplianceCheckResultDataStore(t *testing.T) {
	suite.Run(t, new(complianceCheckResultDataStoreTestSuite))
}

type complianceCheckResultDataStoreTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	hasReadCtx  context.Context
	hasWriteCtx context.Context
	noAccessCtx context.Context

	dataStore DataStore
	storage   checkResultsStorage.Store
	db        *pgtest.TestPostgres
}

func (s *complianceCheckResultDataStoreTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip tests when ComplianceEnhancements disabled")
		s.T().SkipNow()
	}
}

func (s *complianceCheckResultDataStoreTestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.ComplianceOperator)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.ComplianceOperator)))
	s.noAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())

	s.mockCtrl = gomock.NewController(s.T())

	s.db = pgtest.ForT(s.T())

	s.storage = checkResultsStorage.New(s.db)
	s.dataStore = New(s.storage)
}

func (s *complianceCheckResultDataStoreTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *complianceCheckResultDataStoreTestSuite) TestUpsertResult() {
	// make sure we have nothing
	checkResultIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(checkResultIDs)

	rec1 := getTestRec(fixtureconsts.Cluster1)
	rec2 := getTestRec(fixtureconsts.Cluster2)
	ids := []string{rec1.GetId(), rec2.GetId()}

	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec1))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec2))

	count, err := s.storage.Count(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Equal(len(ids), count)

	// upsert with read context
	s.Require().Error(s.dataStore.UpsertResult(s.hasReadCtx, rec2))

	retrieveRec1, found, err := s.storage.Get(s.hasReadCtx, rec1.GetId())
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(rec1, retrieveRec1)
}

func (s *complianceCheckResultDataStoreTestSuite) TestDeleteResult() {
	// make sure we have nothing
	checkResultIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(checkResultIDs)

	rec1 := getTestRec(fixtureconsts.Cluster1)
	rec2 := getTestRec(fixtureconsts.Cluster2)
	ids := []string{rec1.GetId(), rec2.GetId()}

	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec1))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec2))

	count, err := s.storage.Count(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Equal(len(ids), count)

	// Try to delete with wrong context
	s.Require().Error(s.dataStore.DeleteResult(s.hasReadCtx, rec1.GetId()))

	// Now delete rec1
	s.Require().NoError(s.dataStore.DeleteResult(s.hasWriteCtx, rec1.GetId()))
	retrieveRec1, found, err := s.storage.Get(s.hasReadCtx, rec1.GetId())
	s.Require().NoError(err)
	s.Require().False(found)
	s.Require().Nil(retrieveRec1)
}

func (s *complianceCheckResultDataStoreTestSuite) TestSearchCheckResults() {
	// make sure we have nothing
	checkResultIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(checkResultIDs)

	rec1 := getTestRec(fixtureconsts.Cluster1)
	rec2 := getTestRec(fixtureconsts.Cluster2)
	rec3 := getTestRec(fixtureconsts.Cluster2)
	rec4 := getTestRec(fixtureconsts.Cluster2)
	ids := []string{rec1.GetId(), rec2.GetId(), rec3.GetId(), rec4.GetId()}

	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec1))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec2))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec3))
	s.Require().NoError(s.dataStore.UpsertResult(s.hasWriteCtx, rec4))

	count, err := s.storage.Count(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Equal(len(ids), count)

	// Search results of fake cluster, should return 0
	searchResults, err := s.dataStore.SearchCheckResults(s.hasReadCtx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, fixtureconsts.ClusterFake2).ProtoQuery())
	s.Require().NoError(err)
	s.Require().Equal(0, len(searchResults))

	// Search results of cluster 2 should return 3 records.
	searchResults, err = s.dataStore.SearchCheckResults(s.hasReadCtx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, fixtureconsts.Cluster2).ProtoQuery())
	s.Require().NoError(err)
	s.Require().Equal(3, len(searchResults))
}

func getTestRec(clusterID string) *storage.ComplianceOperatorCheckResultV2 {
	return &storage.ComplianceOperatorCheckResultV2{
		Id:           uuid.NewV4().String(),
		CheckId:      uuid.NewV4().String(),
		CheckName:    "test-check",
		ClusterId:    clusterID,
		Status:       storage.ComplianceOperatorCheckResultV2_INCONSISTENT,
		Severity:     storage.RuleSeverity_HIGH_RULE_SEVERITY,
		Description:  "this is a test",
		Instructions: "this is a test",
		Labels:       nil,
		Annotations:  nil,
		CreatedTime:  types.TimestampNow(),
		ScanId:       uuid.NewV4().String(),
	}
}
