//go:build sql_integration

package m200tom201

import (
	"context"
	"testing"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/storage"
	oldSchemas "github.com/stackrox/rox/migrator/migrations/m_200_to_m_201_compliance_v2_for_4_5/test/schema"
	oldProfileStore "github.com/stackrox/rox/migrator/migrations/m_200_to_m_201_compliance_v2_for_4_5/test/stores/profiles"
	oldResultsStore "github.com/stackrox/rox/migrator/migrations/m_200_to_m_201_compliance_v2_for_4_5/test/stores/results"
	oldRuleStore "github.com/stackrox/rox/migrator/migrations/m_200_to_m_201_compliance_v2_for_4_5/test/stores/rules"
	oldScanStore "github.com/stackrox/rox/migrator/migrations/m_200_to_m_201_compliance_v2_for_4_5/test/stores/scans"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

var (
	v2StorageRules = []*storage.ComplianceOperatorProfileV2_Rule{
		{
			RuleName: "rule-1",
		},
		{
			RuleName: "rule-2",
		},
		{
			RuleName: "rule-3",
		},
	}

	fixes = []*storage.ComplianceOperatorRuleV2_Fix{
		{
			Platform:   "openshift",
			Disruption: "its broken",
		},
	}

	profileIDs = []string{uuid.NewV4().String(), uuid.NewV4().String(), uuid.NewV4().String()}
	ruleIDs    = []string{uuid.NewV4().String(), uuid.NewV4().String(), uuid.NewV4().String()}
	resultIDs  = []string{uuid.NewV4().String(), uuid.NewV4().String(), uuid.NewV4().String()}
	scansIDs   = []string{uuid.NewV4().String(), uuid.NewV4().String(), uuid.NewV4().String()}

	startTime = protocompat.TimestampNow()
	endTime   = protocompat.TimestampNow()
)

type migrationTestSuite struct {
	suite.Suite

	db  *pghelper.TestPostgres
	ctx context.Context
	dbs *types.Databases
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)
	s.dbs = &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), oldSchemas.CreateTableComplianceOperatorRuleV2Stmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), oldSchemas.CreateTableComplianceOperatorScanV2Stmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), oldSchemas.CreateTableComplianceOperatorCheckResultV2Stmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), oldSchemas.CreateTableComplianceOperatorProfileV2Stmt)
}

func (s *migrationTestSuite) TestMigration() {
	s.addOldData()

	s.Require().NoError(migration.Run(s.dbs))

	s.compareAfter()
}

func (s *migrationTestSuite) addOldData() {
	profileStore := oldProfileStore.New(s.db.DB)
	s.Require().NoError(profileStore.UpsertMany(s.ctx, getOldProfiles()))

	ruleStore := oldRuleStore.New(s.db.DB)
	s.Require().NoError(ruleStore.UpsertMany(s.ctx, getOldRules()))

	scanStore := oldScanStore.New(s.db.DB)
	s.Require().NoError(scanStore.UpsertMany(s.ctx, getOldScans()))

	resultStore := oldResultsStore.New(s.db.DB)
	s.Require().NoError(resultStore.UpsertMany(s.ctx, getOldCheckResults()))
}

func (s *migrationTestSuite) compareAfter() {
	profileStore := oldProfileStore.New(s.db.DB)
	profiles, _, err := profileStore.GetMany(s.ctx, profileIDs)
	s.Require().NoError(err)
	protoassert.ElementsMatch(s.T(), getExpectedProfiles(), profiles)

	ruleStore := oldRuleStore.New(s.db.DB)
	rules, _, err := ruleStore.GetMany(s.ctx, ruleIDs)
	s.Require().NoError(err)
	protoassert.ElementsMatch(s.T(), getExpectedRules(), rules)

	scanStore := oldScanStore.New(s.db.DB)
	scans, _, err := scanStore.GetMany(s.ctx, scansIDs)
	s.Require().NoError(err)
	protoassert.ElementsMatch(s.T(), getExpectedScans(), scans)

	resultStore := oldResultsStore.New(s.db.DB)
	results, _, err := resultStore.GetMany(s.ctx, resultIDs)
	s.Require().NoError(err)
	protoassert.ElementsMatch(s.T(), getExpectedCheckResults(), results)
}

func getOldProfiles() []*storage.ComplianceOperatorProfileV2 {
	// No Profile Ref ID.  All else the same
	return []*storage.ComplianceOperatorProfileV2{
		{
			Id:             profileIDs[0],
			ProfileId:      "xxx-profile-id-1",
			Name:           "ocp-cis",
			ProfileVersion: "4.2",
			Description:    "this is a test",
			Labels:         nil,
			Annotations:    nil,
			Rules:          v2StorageRules,
			Title:          "Openshift CIS testing",
			ProductType:    "Node",
			Standard:       "",
			Product:        "",
			Values:         nil,
			ClusterId:      fixtureconsts.Cluster1,
		},
		{
			Id:             profileIDs[1],
			ProfileId:      "xxx-profile-id-1",
			Name:           "ocp-cis",
			ProfileVersion: "4.2",
			Description:    "this is a test",
			Labels:         nil,
			Annotations:    nil,
			Rules:          v2StorageRules,
			Title:          "Openshift CIS testing",
			ProductType:    "Node",
			Standard:       "",
			Product:        "",
			Values:         nil,
			ClusterId:      fixtureconsts.Cluster2,
		},
		{
			Id:             profileIDs[2],
			ProfileId:      "xxx-profile-id-2",
			Name:           "ocp-cis",
			ProfileVersion: "4.2",
			Description:    "this is a test",
			Labels:         nil,
			Annotations:    nil,
			Rules:          v2StorageRules,
			Title:          "Openshift CIS testing",
			ProductType:    "Platform",
			Standard:       "",
			Product:        "",
			Values:         nil,
			ClusterId:      fixtureconsts.Cluster1,
		},
	}
}

func getExpectedProfiles() []*storage.ComplianceOperatorProfileV2 {
	return []*storage.ComplianceOperatorProfileV2{
		{
			Id:             profileIDs[0],
			ProfileId:      "xxx-profile-id-1",
			Name:           "ocp-cis",
			ProfileVersion: "4.2",
			Description:    "this is a test",
			Labels:         nil,
			Annotations:    nil,
			Rules:          v2StorageRules,
			Title:          "Openshift CIS testing",
			ProductType:    "Node",
			Standard:       "",
			Product:        "",
			Values:         nil,
			ClusterId:      fixtureconsts.Cluster1,
			ProfileRefId:   createProfileRefID(fixtureconsts.Cluster1, "xxx-profile-id-1", "Node"),
		},
		{
			Id:             profileIDs[1],
			ProfileId:      "xxx-profile-id-1",
			Name:           "ocp-cis",
			ProfileVersion: "4.2",
			Description:    "this is a test",
			Labels:         nil,
			Annotations:    nil,
			Rules:          v2StorageRules,
			Title:          "Openshift CIS testing",
			ProductType:    "Node",
			Standard:       "",
			Product:        "",
			Values:         nil,
			ClusterId:      fixtureconsts.Cluster2,
			ProfileRefId:   createProfileRefID(fixtureconsts.Cluster2, "xxx-profile-id-1", "Node"),
		},
		{
			Id:             profileIDs[2],
			ProfileId:      "xxx-profile-id-2",
			Name:           "ocp-cis",
			ProfileVersion: "4.2",
			Description:    "this is a test",
			Labels:         nil,
			Annotations:    nil,
			Rules:          v2StorageRules,
			Title:          "Openshift CIS testing",
			ProductType:    "Platform",
			Standard:       "",
			Product:        "",
			Values:         nil,
			ClusterId:      fixtureconsts.Cluster1,
			ProfileRefId:   createProfileRefID(fixtureconsts.Cluster1, "xxx-profile-id-2", "Platform"),
		},
	}
}

func getOldRules() []*storage.ComplianceOperatorRuleV2 {
	// Controls have 1 row per standard vs new which has 1 row per standard/control combination
	// rule_ref_id, parent_rule, and instructions (instructions were not carried over and will be empty) do not exist
	controls := []*storage.RuleControls{
		{
			Standard: "NERC-CIP",
			Controls: []string{"CIP-003-8 R6", "CIP-004-6 R3", "CIP-007-3 R6.1"},
		},
		{
			Standard: "PCI-DSS",
			Controls: []string{"Req-2.2"},
		},
	}

	return []*storage.ComplianceOperatorRuleV2{
		{
			Id:          ruleIDs[0],
			RuleId:      "rule-1",
			Name:        "ocp-cis",
			RuleType:    "node",
			Labels:      map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
			Annotations: getAnnotations(),
			Title:       "test rule",
			Description: "test description",
			Rationale:   "test rationale",
			Fixes:       fixes,
			Warning:     "test warning",
			Controls:    controls,
			ClusterId:   fixtureconsts.Cluster1,
		},
		{
			Id:          ruleIDs[1],
			RuleId:      "rule-1",
			Name:        "ocp-cis",
			RuleType:    "node",
			Labels:      map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
			Annotations: getAnnotations(),
			Title:       "test rule",
			Description: "test description",
			Rationale:   "test rationale",
			Fixes:       fixes,
			Warning:     "test warning",
			Controls:    controls,
			ClusterId:   fixtureconsts.Cluster2,
		},
		{
			Id:          ruleIDs[2],
			RuleId:      "rule-2",
			Name:        "ocp-cis",
			RuleType:    "node",
			Labels:      map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
			Annotations: getAnnotations(),
			Title:       "test rule",
			Description: "test description",
			Rationale:   "test rationale",
			Fixes:       fixes,
			Warning:     "test warning",
			Controls:    controls,
			ClusterId:   fixtureconsts.Cluster1,
		},
	}
}

func getExpectedRules() []*storage.ComplianceOperatorRuleV2 {
	controls := []*storage.RuleControls{
		{
			Standard: "NERC-CIP",
			Control:  "CIP-003-8 R6",
		},
		{
			Standard: "NERC-CIP",
			Control:  "CIP-004-6 R3",
		},
		{
			Standard: "NERC-CIP",
			Control:  "CIP-007-3 R6.1",
		},
		{
			Standard: "PCI-DSS",
			Control:  "Req-2.2",
		},
	}

	return []*storage.ComplianceOperatorRuleV2{
		{
			Id:          ruleIDs[0],
			RuleId:      "rule-1",
			Name:        "ocp-cis",
			RuleType:    "node",
			Labels:      map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
			Annotations: getAnnotations(),
			Title:       "test rule",
			Description: "test description",
			Rationale:   "test rationale",
			Fixes:       fixes,
			Warning:     "test warning",
			Controls:    controls,
			ClusterId:   fixtureconsts.Cluster1,
			RuleRefId:   buildDeterministicID(fixtureconsts.Cluster1, "random-rule-name"),
			ParentRule:  "random-rule-name",
		},
		{
			Id:          ruleIDs[1],
			RuleId:      "rule-1",
			Name:        "ocp-cis",
			RuleType:    "node",
			Labels:      map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
			Annotations: getAnnotations(),
			Title:       "test rule",
			Description: "test description",
			Rationale:   "test rationale",
			Fixes:       fixes,
			Warning:     "test warning",
			Controls:    controls,
			ClusterId:   fixtureconsts.Cluster2,
			RuleRefId:   buildDeterministicID(fixtureconsts.Cluster2, "random-rule-name"),
			ParentRule:  "random-rule-name",
		},
		{
			Id:          ruleIDs[2],
			RuleId:      "rule-2",
			Name:        "ocp-cis",
			RuleType:    "node",
			Labels:      map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
			Annotations: getAnnotations(),
			Title:       "test rule",
			Description: "test description",
			Rationale:   "test rationale",
			Fixes:       fixes,
			Warning:     "test warning",
			Controls:    controls,
			ClusterId:   fixtureconsts.Cluster1,
			RuleRefId:   buildDeterministicID(fixtureconsts.Cluster1, "random-rule-name"),
			ParentRule:  "random-rule-name",
		},
	}
}

func getOldScans() []*storage.ComplianceOperatorScanV2 {
	// scan_ref_id and product_type were added.
	// profile_id stays but a profile_ref_id is added
	return []*storage.ComplianceOperatorScanV2{
		{
			Id:             scansIDs[0],
			ScanConfigName: "ocp-cis",
			ScanName:       "ocp-cis",
			ClusterId:      fixtureconsts.Cluster1,
			Errors:         "",
			Warnings:       "",
			Profile: &storage.ProfileShim{
				ProfileId: "xxx-profile-id-1",
			},
			Labels:       map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
			Annotations:  nil,
			ScanType:     storage.ScanType_PLATFORM_SCAN,
			NodeSelector: 0,
			Status: &storage.ScanStatus{
				Phase:    "",
				Result:   "FAIL",
				Warnings: "",
			},
			CreatedTime:      startTime,
			LastExecutedTime: endTime,
		},
		{
			Id:             scansIDs[1],
			ScanConfigName: "ocp-cis",
			ScanName:       "ocp-cis",
			ClusterId:      fixtureconsts.Cluster2,
			Errors:         "",
			Warnings:       "",
			Profile: &storage.ProfileShim{
				ProfileId: "xxx-profile-id-1",
			},
			Labels:       map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
			Annotations:  nil,
			ScanType:     storage.ScanType_PLATFORM_SCAN,
			NodeSelector: 0,
			Status: &storage.ScanStatus{
				Phase:    "",
				Result:   "FAIL",
				Warnings: "",
			},
			CreatedTime:      startTime,
			LastExecutedTime: endTime,
		},
		{
			Id:             scansIDs[2],
			ScanConfigName: "ocp-cis",
			ScanName:       "ocp-cis",
			ClusterId:      fixtureconsts.Cluster1,
			Errors:         "",
			Warnings:       "",
			Profile: &storage.ProfileShim{
				ProfileId: "xxx-profile-id-2",
			},
			Labels:       map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
			Annotations:  nil,
			ScanType:     storage.ScanType_NODE_SCAN,
			NodeSelector: 0,
			Status: &storage.ScanStatus{
				Phase:    "",
				Result:   "FAIL",
				Warnings: "",
			},
			CreatedTime:      startTime,
			LastExecutedTime: endTime,
		},
	}
}

func getExpectedScans() []*storage.ComplianceOperatorScanV2 {
	// scan_ref_id and product_type were added.
	// profile_id stays but a profile_ref_id is added
	return []*storage.ComplianceOperatorScanV2{
		{
			Id:             scansIDs[0],
			ScanConfigName: "ocp-cis",
			ScanName:       "ocp-cis",
			ClusterId:      fixtureconsts.Cluster1,
			Errors:         "",
			Warnings:       "",
			Profile: &storage.ProfileShim{
				ProfileId:    "xxx-profile-id-1",
				ProfileRefId: createProfileRefID(fixtureconsts.Cluster1, "xxx-profile-id-1", "Platform"),
			},
			Labels:       map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
			Annotations:  nil,
			ScanType:     storage.ScanType_PLATFORM_SCAN,
			NodeSelector: 0,
			Status: &storage.ScanStatus{
				Phase:    "",
				Result:   "FAIL",
				Warnings: "",
			},
			CreatedTime:      startTime,
			LastExecutedTime: endTime,
			ProductType:      "Platform",
			ScanRefId:        buildDeterministicID(fixtureconsts.Cluster1, "ocp-cis"),
		},
		{
			Id:             scansIDs[1],
			ScanConfigName: "ocp-cis",
			ScanName:       "ocp-cis",
			ClusterId:      fixtureconsts.Cluster2,
			Errors:         "",
			Warnings:       "",
			Profile: &storage.ProfileShim{
				ProfileId:    "xxx-profile-id-1",
				ProfileRefId: createProfileRefID(fixtureconsts.Cluster2, "xxx-profile-id-1", "Platform"),
			},
			Labels:       map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
			Annotations:  nil,
			ScanType:     storage.ScanType_PLATFORM_SCAN,
			NodeSelector: 0,
			Status: &storage.ScanStatus{
				Phase:    "",
				Result:   "FAIL",
				Warnings: "",
			},
			CreatedTime:      startTime,
			LastExecutedTime: endTime,
			ProductType:      "Platform",
			ScanRefId:        buildDeterministicID(fixtureconsts.Cluster2, "ocp-cis"),
		},
		{
			Id:             scansIDs[2],
			ScanConfigName: "ocp-cis",
			ScanName:       "ocp-cis",
			ClusterId:      fixtureconsts.Cluster1,
			Errors:         "",
			Warnings:       "",
			Profile: &storage.ProfileShim{
				ProfileId:    "xxx-profile-id-2",
				ProfileRefId: createProfileRefID(fixtureconsts.Cluster1, "xxx-profile-id-2", "Node"),
			},
			Labels:       map[string]string{v1alpha1.SuiteLabel: "ocp-cis"},
			Annotations:  nil,
			ScanType:     storage.ScanType_NODE_SCAN,
			NodeSelector: 0,
			Status: &storage.ScanStatus{
				Phase:    "",
				Result:   "FAIL",
				Warnings: "",
			},
			CreatedTime:      startTime,
			LastExecutedTime: endTime,
			ProductType:      "Node",
			ScanRefId:        buildDeterministicID(fixtureconsts.Cluster1, "ocp-cis"),
		},
	}
}

func getOldCheckResults() []*storage.ComplianceOperatorCheckResultV2 {
	// scan_ref_id and rule_ref_id are added.  2 other fields were simply promoted to columns.
	return []*storage.ComplianceOperatorCheckResultV2{
		{
			Id:             resultIDs[0],
			CheckId:        "check-id-1",
			CheckName:      "check-name-1",
			ClusterId:      fixtureconsts.Cluster1,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 1",
			Instructions:   "instructions 1",
			Annotations:    getAnnotations(),
			CreatedTime:    startTime,
			ScanName:       "scan-1",
			ScanConfigName: "scan-1",
			Rationale:      "test rationale",
		},
		{
			Id:             resultIDs[1],
			CheckId:        "check-id-1",
			CheckName:      "check-name-1",
			ClusterId:      fixtureconsts.Cluster2,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 1",
			Instructions:   "instructions 1",
			Annotations:    getAnnotations(),
			CreatedTime:    startTime,
			ScanName:       "scan-1",
			ScanConfigName: "scan-1",
			Rationale:      "test rationale",
		},
		{
			Id:             resultIDs[2],
			CheckId:        "check-id-2",
			CheckName:      "check-name-2",
			ClusterId:      fixtureconsts.Cluster1,
			Status:         storage.ComplianceOperatorCheckResultV2_FAIL,
			Description:    "description 2",
			Instructions:   "instructions 2",
			Annotations:    getAnnotations(),
			CreatedTime:    startTime,
			ScanName:       "scan-2",
			ScanConfigName: "scan-2",
			Rationale:      "test rationale",
		},
	}
}

func getExpectedCheckResults() []*storage.ComplianceOperatorCheckResultV2 {
	// scan_ref_id and rule_ref_id are added.  2 other fields were simply promoted to columns.
	return []*storage.ComplianceOperatorCheckResultV2{
		{
			Id:             resultIDs[0],
			CheckId:        "check-id-1",
			CheckName:      "check-name-1",
			ClusterId:      fixtureconsts.Cluster1,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 1",
			Instructions:   "instructions 1",
			Annotations:    getAnnotations(),
			CreatedTime:    startTime,
			ScanName:       "scan-1",
			ScanConfigName: "scan-1",
			Rationale:      "test rationale",
			ScanRefId:      buildDeterministicID(fixtureconsts.Cluster1, "scan-1"),
			RuleRefId:      buildDeterministicID(fixtureconsts.Cluster1, getAnnotations()[v1alpha1.RuleIDAnnotationKey]),
		},
		{
			Id:             resultIDs[1],
			CheckId:        "check-id-1",
			CheckName:      "check-name-1",
			ClusterId:      fixtureconsts.Cluster2,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 1",
			Instructions:   "instructions 1",
			Annotations:    getAnnotations(),
			CreatedTime:    startTime,
			ScanName:       "scan-1",
			ScanConfigName: "scan-1",
			Rationale:      "test rationale",
			ScanRefId:      buildDeterministicID(fixtureconsts.Cluster2, "scan-1"),
			RuleRefId:      buildDeterministicID(fixtureconsts.Cluster2, getAnnotations()[v1alpha1.RuleIDAnnotationKey]),
		},
		{
			Id:             resultIDs[2],
			CheckId:        "check-id-2",
			CheckName:      "check-name-2",
			ClusterId:      fixtureconsts.Cluster1,
			Status:         storage.ComplianceOperatorCheckResultV2_FAIL,
			Description:    "description 2",
			Instructions:   "instructions 2",
			Annotations:    getAnnotations(),
			CreatedTime:    startTime,
			ScanName:       "scan-2",
			ScanConfigName: "scan-2",
			Rationale:      "test rationale",
			ScanRefId:      buildDeterministicID(fixtureconsts.Cluster1, "scan-2"),
			RuleRefId:      buildDeterministicID(fixtureconsts.Cluster1, getAnnotations()[v1alpha1.RuleIDAnnotationKey]),
		},
	}
}

func getAnnotations() map[string]string {
	annotations := make(map[string]string, 3)
	annotations["policies.open-cluster-management.io/standards"] = "NERC-CIP,PCI-DSS"
	annotations["control.compliance.openshift.io/NERC-CIP"] = "CIP-003-8 R6;CIP-004-6 R3;CIP-007-3 R6.1"
	annotations["control.compliance.openshift.io/PCI-DSS"] = "Req-2.2"
	annotations["compliance.openshift.io/rule"] = "random-rule-name"

	return annotations
}
