//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	benchmarkStorage "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	uuidStub1       = "933cf32f-d387-4787-8835-65857b5fdbfd"
	uuidStub2       = "8f9850a8-b615-4a12-a3da-7b057bf3aeba"
	uuidNonExisting = "1e52b778-63f2-4eab-aa81-c9b6381ceb02"
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
			sac.ResourceScopeKeys(resources.Compliance)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.noAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())

	s.mockCtrl = gomock.NewController(s.T())

	s.db = pgtest.ForT(s.T())
	s.storage = benchmarkStorage.New(s.db)
	s.datastore = New(s.storage)
}

func (s *complianceBenchmarkDataStoreSuite) TestUpsertBenchmark() {
	// make sure we have nothing
	benchmarkIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(benchmarkIDs)

	b1 := getTestBenchmark(uuidStub1, "b1", "1.0", 1)
	b2 := getTestBenchmark(uuidStub2, "b2", "2.0", 2)

	s.Require().NoError(s.datastore.UpsertBenchmark(s.hasWriteCtx, b1))
	s.Require().NoError(s.datastore.UpsertBenchmark(s.hasWriteCtx, b2))

	count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(2, count)

	s.Require().Error(s.datastore.UpsertBenchmark(s.hasReadCtx, b1))

	retB1, found, err := s.storage.Get(s.hasReadCtx, b1.GetId())
	s.Require().NoError(err)
	s.Require().True(found)
	assertBenchmarks(s.T(), b1, retB1)

	retB2, found, err := s.storage.Get(s.hasReadCtx, b2.GetId())
	s.Require().NoError(err)
	s.Require().True(found)
	assertBenchmarks(s.T(), b2, retB2)
}

func (s *complianceBenchmarkDataStoreSuite) TestDeleteBenchmark() {
	// make sure we have nothing
	benchmarkIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(benchmarkIDs)

	b1 := getTestBenchmark(uuidStub1, "b1", "1.0", 1)
	b2 := getTestBenchmark(uuidStub2, "b2", "2.0", 2)

	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, b1))
	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, b2))

	count, err := s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(2, count)

	s.Require().NoError(s.datastore.DeleteBenchmark(s.hasWriteCtx, b1.GetId()))

	count, err = s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(1, count)

	s.Require().NoError(s.datastore.DeleteBenchmark(s.noAccessCtx, b2.GetId()))

	count, err = s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(1, count)

	s.Require().NoError(s.datastore.DeleteBenchmark(s.hasWriteCtx, b2.GetId()))

	count, err = s.storage.Count(s.hasReadCtx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Require().Equal(0, count)
}

func (s *complianceBenchmarkDataStoreSuite) TestGetBenchmark() {
	// make sure we have nothing
	benchmarkIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(benchmarkIDs)

	benchmarks := []*storage.ComplianceOperatorBenchmarkV2{
		getTestBenchmark(uuidStub1, "b1", "1.0", 1),
		getTestBenchmark(uuidStub2, "b2", "2.0", 2),
	}

	for _, b := range benchmarks {
		s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, b))
	}

	for _, b := range benchmarks {
		retB, found, err := s.datastore.GetBenchmark(s.hasReadCtx, b.GetId())
		s.Require().NoError(err)
		s.Require().True(found)
		assertBenchmarks(s.T(), b, retB)
	}

	_, found, err := s.datastore.GetBenchmark(s.noAccessCtx, benchmarks[0].GetId())
	s.Require().NoError(err)
	s.Require().False(found)

	_, found, err = s.datastore.GetBenchmark(s.hasReadCtx, uuidNonExisting)
	s.Require().NoError(err)
	s.Require().False(found)
}

func (s *complianceBenchmarkDataStoreSuite) TestGetBenchmarksByProfileName() {
	benchmark := getTestBenchmark(uuidStub1, "OpenShift CIS", "1.0", 1)
	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, benchmark))

	benchmarks, err := s.datastore.GetBenchmarksByProfileName(s.hasReadCtx, benchmark.GetProfiles()[0].GetProfileName())
	s.Require().NoError(err)
	s.Require().Len(benchmarks, 1)
	s.Assert().Equal("OpenShift CIS", benchmarks[0].Name)
}

func getTestBenchmark(id string, name string, version string, profileCount int) *storage.ComplianceOperatorBenchmarkV2 {
	var profiles []*storage.ComplianceOperatorBenchmarkV2_Profile
	for i := 0; i < profileCount; i++ {
		profiles = append(profiles, &storage.ComplianceOperatorBenchmarkV2_Profile{
			ProfileName:    fmt.Sprintf("%s-%d", name, i),
			ProfileVersion: fmt.Sprintf("%s-%d", version, i),
		})
	}
	return &storage.ComplianceOperatorBenchmarkV2{
		Id:        id,
		Name:      name,
		Version:   version,
		Provider:  fmt.Sprintf("provider-%s", name),
		Profiles:  profiles,
		ShortName: fmt.Sprintf("short-name-%s", name),
	}
}

func assertBenchmarks(t *testing.T, expected *storage.ComplianceOperatorBenchmarkV2, actual *storage.ComplianceOperatorBenchmarkV2) {
	assert.Equal(t, expected.GetId(), actual.GetId())
	assert.Equal(t, expected.GetName(), actual.GetName())
	assert.Equal(t, expected.GetVersion(), actual.GetVersion())
	assert.Equal(t, expected.GetDescription(), actual.GetDescription())
	assert.Equal(t, expected.GetProvider(), actual.GetProvider())
	assert.Equal(t, expected.GetShortName(), actual.GetShortName())
	assert.Equal(t, len(expected.GetProfiles()), len(actual.GetProfiles()))
	for _, p := range expected.GetProfiles() {
		protoassert.SliceContains(t, actual.GetProfiles(), p)
	}
}
