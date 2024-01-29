//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	ruleStorage "github.com/stackrox/rox/central/complianceoperator/v2/rules/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestComplianceRuleDataStore(t *testing.T) {
	suite.Run(t, new(complianceRuleDataStoreTestSuite))
}

type complianceRuleDataStoreTestSuite struct {
	suite.Suite

	hasReadCtx            context.Context
	hasWriteCtx           context.Context
	noAccessCtx           context.Context
	testContexts          map[string]context.Context
	nonComplianceContexts map[string]context.Context

	dataStore DataStore
	storage   ruleStorage.Store
	db        *pgtest.TestPostgres
}

func (s *complianceRuleDataStoreTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip tests when ComplianceEnhancements disabled")
		s.T().SkipNow()
	}
}

func (s *complianceRuleDataStoreTestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.noAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Compliance)
	s.nonComplianceContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Deployment)

	s.db = pgtest.ForT(s.T())

	s.storage = ruleStorage.New(s.db)
	s.dataStore = GetTestPostgresDataStore(s.T(), s.db)
}

func (s *complianceRuleDataStoreTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *complianceRuleDataStoreTestSuite) TestUpsertRule() {
	// make sure we have nothing
	ruleIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(ruleIDs)

	testCases := []struct {
		desc                string
		rules               []*storage.ComplianceOperatorRuleV2
		testContext         context.Context
		expectedRecordIndex set.FrozenIntSet
	}{
		{
			desc: "Write 3 clusters - Full access",
			rules: []*storage.ComplianceOperatorRuleV2{
				getTestRule(testconsts.Cluster1),
				getTestRule(testconsts.Cluster2),
				getTestRule(testconsts.Cluster3),
			},
			testContext:         s.testContexts[testutils.UnrestrictedReadWriteCtx],
			expectedRecordIndex: set.NewFrozenIntSet(0, 1, 2),
		},
		{
			desc: "Write 3 clusters - No access",
			rules: []*storage.ComplianceOperatorRuleV2{
				getTestRule(testconsts.Cluster1),
				getTestRule(testconsts.Cluster2),
				getTestRule(testconsts.Cluster3),
			},
			testContext:         s.noAccessCtx,
			expectedRecordIndex: set.NewFrozenIntSet(),
		},
		{
			desc: "Write 3 clusters - Cluster 1 access",
			rules: []*storage.ComplianceOperatorRuleV2{
				getTestRule(testconsts.Cluster1),
				getTestRule(testconsts.Cluster2),
				getTestRule(testconsts.Cluster3),
			},
			testContext:         s.testContexts[testutils.Cluster1ReadWriteCtx],
			expectedRecordIndex: set.NewFrozenIntSet(0),
		},
	}

	for _, tc := range testCases {
		for index, rule := range tc.rules {
			if tc.expectedRecordIndex.Contains(index) {
				s.Require().NoError(s.dataStore.UpsertRule(tc.testContext, rule))
			} else {
				s.Require().Error(s.dataStore.UpsertRule(tc.testContext, rule), "access to resource denied")
			}
		}

		count, err := s.storage.Count(s.hasReadCtx)
		s.Require().NoError(err)
		s.Require().Equal(tc.expectedRecordIndex.Cardinality(), count)

		// Clean up
		for _, rule := range tc.rules {
			s.Require().NoError(s.dataStore.DeleteRule(s.hasWriteCtx, rule.GetId()))
		}
	}
}

func (s *complianceRuleDataStoreTestSuite) TestDeleteRule() {
	// make sure we have nothing
	ruleIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(ruleIDs)

	testCases := []struct {
		desc                string
		rules               []*storage.ComplianceOperatorRuleV2
		testContext         context.Context
		expectedRecordIndex set.FrozenIntSet
	}{
		{
			desc: "Write 3 clusters - Full access",
			rules: []*storage.ComplianceOperatorRuleV2{
				getTestRule(testconsts.Cluster1),
				getTestRule(testconsts.Cluster2),
				getTestRule(testconsts.Cluster3),
			},
			testContext:         s.testContexts[testutils.UnrestrictedReadWriteCtx],
			expectedRecordIndex: set.NewFrozenIntSet(0, 1, 2),
		},
		{
			desc: "Write 3 clusters - No access",
			rules: []*storage.ComplianceOperatorRuleV2{
				getTestRule(testconsts.Cluster1),
				getTestRule(testconsts.Cluster2),
				getTestRule(testconsts.Cluster3),
			},
			testContext:         s.noAccessCtx,
			expectedRecordIndex: set.NewFrozenIntSet(),
		},
		{
			desc: "Write 3 clusters - Cluster 1 access",
			rules: []*storage.ComplianceOperatorRuleV2{
				getTestRule(testconsts.Cluster1),
				getTestRule(testconsts.Cluster2),
				getTestRule(testconsts.Cluster3),
			},
			testContext:         s.testContexts[testutils.Cluster1ReadWriteCtx],
			expectedRecordIndex: set.NewFrozenIntSet(0),
		},
	}

	for _, tc := range testCases {
		for _, rule := range tc.rules {
			s.Require().NoError(s.dataStore.UpsertRule(s.hasWriteCtx, rule))
		}

		for _, rule := range tc.rules {
			s.Require().NoError(s.dataStore.DeleteRule(tc.testContext, rule.GetId()))
		}

		count, err := s.storage.Count(s.hasReadCtx)
		s.Require().NoError(err)
		// If we could not delete the rules then they will remain.
		s.Require().Equal(len(tc.rules)-tc.expectedRecordIndex.Cardinality(), count)

		// Clean up
		for _, rule := range tc.rules {
			s.Require().NoError(s.dataStore.DeleteRule(s.hasWriteCtx, rule.GetId()))
		}
	}
}

func (s *complianceRuleDataStoreTestSuite) TestGetRulesByCluster() {
	// make sure we have nothing
	ruleIDs, err := s.storage.GetIDs(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Empty(ruleIDs)

	testRule1 := getTestRule(testconsts.Cluster1)
	testRule2 := getTestRule(testconsts.Cluster1)
	testRule3 := getTestRule(testconsts.Cluster2)

	s.Require().NoError(s.dataStore.UpsertRule(s.hasWriteCtx, testRule1))
	s.Require().NoError(s.dataStore.UpsertRule(s.hasWriteCtx, testRule2))
	s.Require().NoError(s.dataStore.UpsertRule(s.hasWriteCtx, testRule3))

	count, err := s.storage.Count(s.hasReadCtx)
	s.Require().NoError(err)
	s.Require().Equal(3, count)

	testCases := []struct {
		desc            string
		clusterID       string
		testContext     context.Context
		expectedResults []*storage.ComplianceOperatorRuleV2
		expectedCount   int
	}{
		{
			desc:            "Rules exist - Full access",
			clusterID:       testconsts.Cluster1,
			testContext:     s.testContexts[testutils.UnrestrictedReadCtx],
			expectedResults: []*storage.ComplianceOperatorRuleV2{testRule1, testRule2},
			expectedCount:   2,
		},
		{
			desc:            "Rules exist - Cluster 1 access",
			clusterID:       testconsts.Cluster1,
			testContext:     s.testContexts[testutils.Cluster1ReadWriteCtx],
			expectedResults: []*storage.ComplianceOperatorRuleV2{testRule1, testRule2},
			expectedCount:   2,
		},
		{
			desc:            "Rules exist - Cluster 2 access",
			clusterID:       testconsts.Cluster1,
			testContext:     s.testContexts[testutils.Cluster2ReadWriteCtx],
			expectedResults: nil,
			expectedCount:   0,
		},
		{
			desc:            "Rules exists - No compliance access",
			clusterID:       testconsts.Cluster1,
			testContext:     s.nonComplianceContexts[testutils.UnrestrictedReadCtx],
			expectedResults: nil,
			expectedCount:   0,
		},
		{
			desc:            "Rule does not exist - Full access",
			clusterID:       fixtureconsts.ClusterFake1,
			testContext:     s.testContexts[testutils.UnrestrictedReadCtx],
			expectedResults: nil,
			expectedCount:   0,
		},
	}
	for _, tc := range testCases {
		retrievedObjects, err := s.dataStore.GetRulesByCluster(tc.testContext, tc.clusterID)
		s.Require().NoError(err)
		s.Require().Equal(tc.expectedCount, len(retrievedObjects))
		s.Require().Equal(tc.expectedResults, retrievedObjects)
	}
}

func getTestRule(clusterID string) *storage.ComplianceOperatorRuleV2 {
	annotations := make(map[string]string, 5)
	annotations["policies.open-cluster-management.io/standards"] = "NERC-CIP,NIST-800-53,PCI-DSS,CIS-OCP"
	annotations["control.compliance.openshift.io/NERC-CIP"] = "CIP-003-8 R6;CIP-004-6 R3;CIP-007-3 R6.1"
	annotations["control.compliance.openshift.io/NIST-800-53"] = "CM-6;CM-6(1)"
	annotations["control.compliance.openshift.io/PCI-DSS"] = "Req-2.2"
	annotations["control.compliance.openshift.io/CIS-OCP"] = "5.1.6"

	fixes := []*storage.ComplianceOperatorRuleV2_Fix{
		{
			Platform:   "openshift",
			Disruption: "its broken",
		},
	}

	controls := []*storage.RuleControls{
		{
			Standard: "CIS",
			Controls: []string{"1.2", "2.3", "1.6.3"},
		},
	}

	return &storage.ComplianceOperatorRuleV2{
		Id:          uuid.NewV4().String(),
		RuleId:      uuid.NewV4().String(),
		Name:        uuid.NewV4().String(),
		RuleType:    "node",
		Severity:    0,
		Labels:      nil,
		Annotations: annotations,
		Title:       "Test rule for cluster " + clusterID,
		Description: "testing",
		Rationale:   "to test",
		Fixes:       fixes,
		Warning:     "",
		Controls:    controls,
		ClusterId:   clusterID,
	}
}
