//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	remediationStorage "github.com/stackrox/rox/central/complianceoperator/v2/remediations/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestComplianceRemediationDataStore(t *testing.T) {
	suite.Run(t, new(complianceRemediationDataStoreTestSuite))
}

type complianceRemediationDataStoreTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	hasReadCtx  context.Context
	hasWriteCtx context.Context

	dataStore DataStore
	storage   remediationStorage.Store
	db        *pgtest.TestPostgres
}

func (s *complianceRemediationDataStoreTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	s.T().Setenv(features.ComplianceRemediationV2.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip tests when ComplianceEnhancements disabled")
		s.T().SkipNow()
	}
}

func (s *complianceRemediationDataStoreTestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))

	s.mockCtrl = gomock.NewController(s.T())

	s.db = pgtest.ForT(s.T())

	s.storage = remediationStorage.New(s.db)
	s.dataStore = GetTestPostgresDataStore(s.T(), s.db)
}

func (s *complianceRemediationDataStoreTestSuite) TestSearchRemediation() {
	remediationFixture := &storage.ComplianceOperatorRemediationV2{
		Id:                        uuid.NewV4().String(),
		ClusterId:                 uuid.NewV4().String(),
		Name:                      "test-name",
		ComplianceCheckResultName: "test-check-res",
	}
	// test insert
	err := s.dataStore.UpsertRemediation(s.hasWriteCtx, remediationFixture)
	s.Require().NoError(err)
	remediationFixture = &storage.ComplianceOperatorRemediationV2{
		Id:                        uuid.NewV4().String(),
		ClusterId:                 uuid.NewV4().String(),
		Name:                      "test-name2",
		ComplianceCheckResultName: "test-check-res2",
	}
	// test insert
	err = s.dataStore.UpsertRemediation(s.hasWriteCtx, remediationFixture)
	s.Require().NoError(err)
	q := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorCheckName, "test-check-res").ProtoQuery()
	remediations, err := s.dataStore.SearchRemediations(s.hasReadCtx, q)
	s.Require().NoError(err)
	s.Require().NotEmpty(remediations)
	s.Require().Equal(remediations[0].GetComplianceCheckResultName(), "test-check-res")

}

func (s *complianceRemediationDataStoreTestSuite) TestRemediation() {
	remediationFixture := &storage.ComplianceOperatorRemediationV2{
		Id:                        uuid.NewV4().String(),
		ClusterId:                 uuid.NewV4().String(),
		Name:                      "some name",
		ComplianceCheckResultName: "some name",
	}

	// test insert
	err := s.dataStore.UpsertRemediation(s.hasWriteCtx, remediationFixture)
	s.Require().NoError(err)

	// test get by ID
	remediationResult, found, err := s.dataStore.GetRemediation(s.hasReadCtx, remediationFixture.GetId())
	s.Require().NoError(err)
	s.Require().True(found, "remediation object should be found")
	protoassert.Equal(s.T(), remediationFixture, remediationResult)

	// test get by cluster ID
	remediationResultByCluster, err := s.dataStore.GetRemediationsByCluster(s.hasReadCtx, remediationFixture.GetClusterId())
	s.Require().NoError(err)
	s.Require().Len(remediationResultByCluster, 1)

	// test delete
	err = s.dataStore.DeleteRemediation(s.hasWriteCtx, remediationFixture.GetId())
	s.Require().NoError(err)
	remediationNotFound, found, err := s.dataStore.GetRemediation(s.hasReadCtx, remediationFixture.GetId())
	s.Require().NoError(err)
	s.Require().False(found, "remediation was not found")
	s.Require().Empty(remediationNotFound)

	// test delete by cluster id
	err = s.dataStore.UpsertRemediation(s.hasWriteCtx, remediationFixture)
	s.Require().NoError(err)
	err = s.dataStore.DeleteRemediationsByCluster(s.hasWriteCtx, remediationFixture.GetClusterId())
	s.Require().NoError(err)
	remediationNotFoundByCluster, err := s.dataStore.GetRemediationsByCluster(s.hasWriteCtx, remediationFixture.GetClusterId())
	s.Require().NoError(err)
	s.Require().Empty(remediationNotFoundByCluster)
}
