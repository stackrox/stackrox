package datastore

import (
	"context"
	"os"
	"testing"

	benchmarkstore "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/benchmarkstore/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestComplianceBenchmarkDataStore(t *testing.T) {
	suite.Run(t, new(complianceIntegrationDataStoreTestSuite))
}

type complianceIntegrationDataStoreTestSuite struct {
	suite.Suite

	hasReadCtx   context.Context
	hasWriteCtx  context.Context
	noAccessCtx  context.Context
	testContexts map[string]context.Context

	dataStore Datastore
	db        *pgtest.TestPostgres
	storage   benchmarkstore.Store
}

func (s *complianceIntegrationDataStoreTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip tests when ComplianceEnhancements disabled")
		s.T().SkipNow()
	}
}

func (s *complianceIntegrationDataStoreTestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.noAccessCtx = sac.WithNoAccess(context.Background())
	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Compliance)

	os.Setenv("POSTGRES_PORT", "54323")
	s.db = pgtest.ForT(s.T())
	s.storage = benchmarkstore.New(s.db)
	s.dataStore = &datastoreImpl{
		benchmarkStore: s.storage,
	}
}

func (s *complianceIntegrationDataStoreTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *complianceIntegrationDataStoreTestSuite) TestAddComplianceIntegration() {
	benchmark := &storage.ComplianceOperatorBenchmark{
		Id:      uuid.NewV4().String(),
		Version: "4.5.0",
		Name:    "CIS OpenShift Benchmark",
	}
	err := s.dataStore.UpsertBenchmark(s.hasWriteCtx, benchmark)
	s.Require().NoError(err)
}
