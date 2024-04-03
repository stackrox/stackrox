package datastore

import (
	"context"
	"os"
	"testing"

	benchmarkstore "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/benchmarkstore/postgres"
	controlstore "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/control_store/postgres"
	controlruleedgestore "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/controlruleedgestore/postgres"
	rulestore "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/rules/store/postgres"
	"github.com/stackrox/rox/pkg/search"
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

	benchmarkStore Datastore
	db             *pgtest.TestPostgres
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
	benchmarkStorage := benchmarkstore.New(s.db)
	controlStore := controlstore.New(s.db)
	compliancestore := pgStore.New(s.db)
	ruleStore := rulestore.New(compliancestore)
	controlruleedgestore := controlruleedgestore.New(s.db)

	s.benchmarkStore = &datastoreImpl{
		benchmarkStore:       benchmarkStorage,
		controlStore:         controlStore,
		ruleStore:            ruleStore,
		controlRuleEdgeStore: controlruleedgestore,
	}
	datastoreSingleton = s.benchmarkStore
}

func (s *complianceIntegrationDataStoreTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

// TODO: Parse correct annotation
// Annotations:
// control.compliance.openshift.io/CIS-OCP
// "policies.open-cluster-management.io/controls": "CIP-003-8 R6,P-004-6 R3,CIP-007-3 R6.1,CM-6,CM-6(1),Req-2.2,1.2.1",
// "policies.open-cluster-management.io/standards": "NERC-CIP,NIST-800-53,PCI-DSS,CIS-OCP"
//
//	 "compliance.openshift.io/image-digest": "pb-ocp4qx7xv",
//	"compliance.openshift.io/profiles": "ocp4-high,ocp4-pci-dss,ocp4-moderate-rev-4,ocp4-stig-v1r1,ocp4-moderate,ocp4-bsi-2022,ocp4-cis,ocp4-high-rev-4,ocp4-nerc-cip,ocp4-cis-1-4,ocp4-pci-dss-3-2,ocp4-stig,ocp4-bsi,ocp4-cis-1-5",
//	"compliance.openshift.io/rule": "api-server-anonymous-auth",
//	"control.compliance.openshift.io/CIS-OCP": "1.2.1",
//	"control.compliance.openshift.io/NERC-CIP": "CIP-003-8 R6;CIP-004-6 R3;CIP-007-3 R6.1",
//	"control.compliance.openshift.io/NIST-800-53": "CM-6;CM-6(1)",
//	"control.compliance.openshift.io/PCI-DSS": "Req-2.2",
//	"policies.open-cluster-management.io/controls": "CIP-003-8 R6,P-004-6 R3,CIP-007-3 R6.1,CM-6,CM-6(1),Req-2.2,1.2.1",
//	"policies.open-cluster-management.io/standards": "NERC-CIP,NIST-800-53,PCI-DSS,CIS-OCP"
func (s *complianceIntegrationDataStoreTestSuite) TestAddBenchmark() {
	benchmark := &storage.ComplianceOperatorBenchmark{
		Id:           uuid.NewV4().String(),
		Version:      "v1.5.0",
		Name:         "CIS Red Hat OpenShift Container Platform Benchmark",
		ProfileLabel: "control.compliance.openshift.io/CIS-OCP",
	}
	err := s.benchmarkStore.UpsertBenchmark(s.hasWriteCtx, benchmark)
	s.Require().NoError(err)
}

func (s *complianceIntegrationDataStoreTestSuite) TestAddControl() {
	benchmark := &storage.ComplianceOperatorBenchmark{
		Id:   uuid.NewDummy().String(),
		Name: "CIS OpenShift",
	}
	err := s.benchmarkStore.UpsertBenchmark(s.hasWriteCtx, benchmark)
	s.Require().NoError(err)

	control := &storage.ComplianceOperatorControl{
		Id:          uuid.NewV4().String(),
		Control:     "1.1.1",
		BenchmarkId: benchmark.GetId(),
	}

	err = s.benchmarkStore.UpsertControl(s.hasWriteCtx, control)
	s.Require().NoError(err)

	controlResult, found, err := s.benchmarkStore.GetControl(s.hasReadCtx, control.GetId())
	s.Require().NoError(err)
	s.True(found)
	s.Equal("1.1.1", controlResult.GetControl())

	rulestore := rulestore.New(pgStore.New(s.db))
	clusterId := uuid.NewV4().String()
	rule := &storage.ComplianceOperatorRuleV2{
		Id:     uuid.NewV4().String(),
		RuleId: uuid.NewV4().String(),
		Name:   "rule from compliance operator",
		Annotations: map[string]string{
			CISBenchmarkAnnotation: "1.1.1",
		},
		ClusterId: clusterId,
	}
	err = rulestore.UpsertRule(s.hasWriteCtx, rule)
	s.Require().NoError(err)

	rulesByClusterId, err := rulestore.GetRulesByCluster(s.hasReadCtx, clusterId)
	s.Require().NoError(err)
	s.Len(rulesByClusterId, 1)

	controlruleedgestore := controlruleedgestore.New(s.db)
	ids, err := controlruleedgestore.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Len(ids, 1)

	query := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorControlRuleRule, rule.GetId()).AddExactMatches(search.ControlID, control.GetId()).ProtoQuery()
	linkResults, err := controlruleedgestore.Search(s.hasReadCtx, query)
	s.Require().NoError(err)
	s.Len(linkResults, 1)
}
