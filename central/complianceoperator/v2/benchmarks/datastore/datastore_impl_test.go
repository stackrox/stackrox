package datastore

import (
	"context"
	"fmt"
	"testing"

	benchmarkStorage "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestComplianceBenchmarkDataStore(t *testing.T) {
	suite.Run(t, new(complianceBenchmarkDataStoreSuite))
}

type complianceBenchmarkDataStoreSuite struct {
	suite.Suite

	hasReadCtx  context.Context
	hasWriteCtx context.Context
	noAccessCtx context.Context

	mockCtrl *gomock.Controller

	datastore DataStore
	storage   benchmarkStorage.Store
	db        *pgtest.TestPostgres
}

func (s *complianceBenchmarkDataStoreSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skipf("Skip test when %s is disabled", features.ComplianceEnhancements.EnvVar())
		s.T().SkipNow()
	}
}

func (s *complianceBenchmarkDataStoreSuite) SetupTest() {
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
	s.T().Setenv("POSTGRES_PORT", "5432")
	s.T().Setenv("POSTGRES_PASSWORD", "password")
	s.T().Setenv("USER", "postgres")

	s.db = pgtest.ForT(s.T())
	s.storage = benchmarkStorage.New(s.db)
	//s.datastore = New(s.storage)
	s.datastore = &datastoreImpl{
		store: s.storage,
		db:    s.db,
	}
}

func (s *complianceBenchmarkDataStoreSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *complianceBenchmarkDataStoreSuite) TestGetControl() {
	result, err := s.datastore.GetControlByRuleId(s.hasReadCtx, []string{"ocp4-api-server-anonymous-auth", "ocp4-api-server-admission-control-plugin-namespacelifecycle"})
	s.Require().NoError(err)
	s.Len(result, -1)
}

func (s *complianceBenchmarkDataStoreSuite) TestUpsertBenchmark() {
	// make sure we have nothing
	benchmarkIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(benchmarkIDs)

	b1 := getTestBenchmark("1", "b1", "1.0", 1)
	b2 := getTestBenchmark("2", "b2", "2.0", 2)

	s.Assert().NoError(s.datastore.UpsertBenchmark(s.hasWriteCtx, b1))
	s.Assert().NoError(s.datastore.UpsertBenchmark(s.hasWriteCtx, b2))

	count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Assert().NoError(err)
	s.Assert().Equal(2, count)

	s.Assert().Error(s.datastore.UpsertBenchmark(s.hasReadCtx, b1))

	retB1, found, err := s.storage.Get(s.hasReadCtx, b1.GetId())
	s.Assert().NoError(err)
	s.Assert().True(found)
	assertBenchmarks(s.T(), b1, retB1)

	retB2, found, err := s.storage.Get(s.hasReadCtx, b2.GetId())
	s.Assert().NoError(err)
	s.Assert().True(found)
	assertBenchmarks(s.T(), b2, retB2)
}

func (s *complianceBenchmarkDataStoreSuite) TestDeleteBenchmark() {
	// make sure we have nothing
	benchmarkIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(benchmarkIDs)

	b1 := getTestBenchmark("1", "b1", "1.0", 1)
	b2 := getTestBenchmark("2", "b2", "2.0", 2)

	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, b1))
	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, b2))

	count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(2, count)

	s.Assert().NoError(s.datastore.DeleteBenchmark(s.hasWriteCtx, b1.GetId()))

	count, err = s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Assert().NoError(err)
	s.Assert().Equal(1, count)

	s.Assert().Error(s.datastore.DeleteBenchmark(s.hasReadCtx, b2.GetId()))

	s.Assert().NoError(s.datastore.DeleteBenchmark(s.hasWriteCtx, b2.GetId()))

	count, err = s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Assert().NoError(err)
	s.Assert().Equal(0, count)
}

func (s *complianceBenchmarkDataStoreSuite) TestGetBenchmark() {
	// make sure we have nothing
	benchmarkIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(benchmarkIDs)

	benchmarks := []*storage.ComplianceOperatorBenchmarkV2{
		getTestBenchmark("1", "b1", "1.0", 1),
		getTestBenchmark("2", "b2", "2.0", 2),
	}

	for _, b := range benchmarks {
		s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, b))
	}

	for _, b := range benchmarks {
		retB, found, err := s.datastore.GetBenchmark(s.hasReadCtx, b.GetId())
		s.Assert().NoError(err)
		s.Assert().True(found)
		assertBenchmarks(s.T(), b, retB)
	}

	_, found, err := s.datastore.GetBenchmark(s.noAccessCtx, benchmarks[0].GetId())
	s.Assert().NoError(err)
	s.Assert().False(found)

	_, found, err = s.datastore.GetBenchmark(s.hasReadCtx, "non-existing-id")
	s.Assert().NoError(err)
	s.Assert().False(found)
}

func (s *complianceBenchmarkDataStoreSuite) TestSearchBenchmarks() {
	// make sure we have nothing
	benchmarkIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(benchmarkIDs)

	b1 := getTestBenchmark("1", "b1", "1.0", 1)
	b2 := getTestBenchmark("2", "b2", "2.0", 2)

	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, b1))
	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, b2))

	retBenchmarks, err := s.storage.Search(s.hasReadCtx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorBenchmarkName, b1.GetName()).ProtoQuery())
	s.Assert().NoError(err)
	s.Assert().Equal(1, len(retBenchmarks))
	s.Assert().Contains(retBenchmarks, b1)

	retBenchmarks, err = s.storage.Search(s.hasReadCtx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorBenchmarkName, "non-existing-name").ProtoQuery())
	s.Assert().NoError(err)
	s.Assert().Equal(0, len(retBenchmarks))

	retBenchmarks, err = s.storage.Search(s.noAccessCtx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorBenchmarkName, b1.GetName()).ProtoQuery())
	s.Assert().NoError(err)
	s.Assert().Empty(retBenchmarks)
}

func getTestBenchmark(id string, name string, version string, profileCount int) *storage.ComplianceOperatorBenchmarkV2 {
	var profiles []*storage.ComplianceOperatorBenchmarkV2_Profile
	for i := 0; i < profileCount; i++ {
		profiles = append(profiles, &storage.ComplianceOperatorBenchmarkV2_Profile{
			ProfileName:       fmt.Sprintf("%s-%d", name, i),
			ProfileVersion:    fmt.Sprintf("%s-%d", version, i),
			ProfileAnnotation: fmt.Sprintf("annotation-%s", name),
		})
	}
	return &storage.ComplianceOperatorBenchmarkV2{
		Id:       id,
		Name:     name,
		Version:  version,
		Provider: fmt.Sprintf("provider-%s", name),
		Profiles: profiles,
	}
}

func assertBenchmarks(t *testing.T, expected *storage.ComplianceOperatorBenchmarkV2, actual *storage.ComplianceOperatorBenchmarkV2) {
	assert.Equal(t, expected.GetId(), actual.GetId())
	assert.Equal(t, expected.GetName(), actual.GetName())
	assert.Equal(t, expected.GetVersion(), actual.GetVersion())
	assert.Equal(t, expected.GetDescription(), actual.GetDescription())
	assert.Equal(t, expected.GetProvider(), actual.GetProvider())
	assert.Equal(t, len(expected.GetProfiles()), len(actual.GetProfiles()))
	for _, p := range expected.GetProfiles() {
		assert.Contains(t, actual.GetProfiles(), p)
	}
}
