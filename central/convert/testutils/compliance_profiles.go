package testutils

import (
	"testing"

	"github.com/stackrox/rox/central/convert/internaltov2storage"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	// ProfileUID -- profile UID used in test objects
	ProfileUID  = uuid.NewV4().String()
	profileUID2 = uuid.NewV4().String()

	RemediationUID = uuid.NewV4().String()

	profileID     = uuid.NewV4().String()
	values        = []string{"value-1", "value-2"}
	v2SensorRules = []*central.ComplianceOperatorProfileV2_Rule{
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

	v1StorageRules = []*storage.ComplianceOperatorProfile_Rule{
		{
			Name: "rule-1",
		},
		{
			Name: "rule-2",
		},
		{
			Name: "rule-3",
		},
	}

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

	v2ApiRules = []*v2.ComplianceRule{
		{
			Name: "rule-1",
		},
		{
			Name: "rule-2",
		},
		{
			Name: "rule-3",
		},
	}
)

// GetProfileV1SensorMsg -- returns a V1 storage object
func GetProfileV1SensorMsg(_ *testing.T) *storage.ComplianceOperatorProfile {
	cop := &storage.ComplianceOperatorProfile{}
	cop.SetId(ProfileUID)
	cop.SetProfileId(profileID)
	cop.SetName("ocp-cis")
	cop.SetClusterId(fixtureconsts.Cluster1)
	cop.SetDescription("this is a test")
	cop.SetLabels(nil)
	cop.SetAnnotations(nil)
	cop.SetRules(v1StorageRules)
	return cop
}

// GetProfileV2SensorMsg -- returns a V2 message from sensor
func GetProfileV2SensorMsg(_ *testing.T) *central.ComplianceOperatorProfileV2 {
	copv2 := &central.ComplianceOperatorProfileV2{}
	copv2.SetId(ProfileUID)
	copv2.SetProfileId(profileID)
	copv2.SetName("ocp-cis")
	copv2.SetProfileVersion("4.2")
	copv2.SetDescription("this is a test")
	copv2.SetLabels(nil)
	copv2.SetAnnotations(nil)
	copv2.SetRules(v2SensorRules)
	copv2.SetTitle("Openshift CIS testing")
	copv2.SetValues(values)
	return copv2
}

// GetProfileV2Storage -- returns a V2 storage object
func GetProfileV2Storage(_ *testing.T) *storage.ComplianceOperatorProfileV2 {
	copv2 := &storage.ComplianceOperatorProfileV2{}
	copv2.SetId(ProfileUID)
	copv2.SetProfileId(profileID)
	copv2.SetName("ocp-cis")
	copv2.SetProfileVersion("4.2")
	copv2.SetDescription("this is a test")
	copv2.SetLabels(nil)
	copv2.SetAnnotations(nil)
	copv2.SetRules(v2StorageRules)
	copv2.SetTitle("Openshift CIS testing")
	copv2.SetProductType("")
	copv2.SetStandard("")
	copv2.SetProduct("")
	copv2.SetValues(values)
	copv2.SetClusterId(fixtureconsts.Cluster1)
	copv2.SetProfileRefId(internaltov2storage.BuildProfileRefID(fixtureconsts.Cluster1, profileID, ""))
	return copv2
}

// GetProfilesV2Storage -- returns a V2 storage object
func GetProfilesV2Storage(_ *testing.T) []*storage.ComplianceOperatorProfileV2 {
	copv2 := &storage.ComplianceOperatorProfileV2{}
	copv2.SetId(ProfileUID)
	copv2.SetProfileId(profileID)
	copv2.SetName("ocp-cis")
	copv2.SetProfileVersion("4.2")
	copv2.SetDescription("this is a test")
	copv2.SetLabels(nil)
	copv2.SetAnnotations(nil)
	copv2.SetRules(v2StorageRules)
	copv2.SetTitle("Openshift CIS testing")
	copv2.SetProductType("")
	copv2.SetStandard("")
	copv2.SetProduct("")
	copv2.SetValues(values)
	copv2.SetClusterId(fixtureconsts.Cluster1)
	copv2.SetProfileRefId(internaltov2storage.BuildProfileRefID(fixtureconsts.Cluster1, profileID, ""))
	copv2h2 := &storage.ComplianceOperatorProfileV2{}
	copv2h2.SetId(profileUID2)
	copv2h2.SetProfileId(profileID)
	copv2h2.SetName("rhcos-moderate")
	copv2h2.SetProfileVersion("4.1.2")
	copv2h2.SetDescription("this is a test")
	copv2h2.SetLabels(nil)
	copv2h2.SetAnnotations(nil)
	copv2h2.SetRules(v2StorageRules)
	copv2h2.SetTitle("Openshift CIS testing")
	copv2h2.SetProductType("")
	copv2h2.SetStandard("")
	copv2h2.SetProduct("")
	copv2h2.SetValues(values)
	copv2h2.SetClusterId(fixtureconsts.Cluster1)
	copv2h2.SetProfileRefId(internaltov2storage.BuildProfileRefID(fixtureconsts.Cluster1, profileID, ""))
	return []*storage.ComplianceOperatorProfileV2{
		copv2,
		copv2h2,
	}
}

// GetProfileV2Api returns a V2 API profile that matches GetProfileV2Storage
func GetProfileV2Api(_ *testing.T) *v2.ComplianceProfile {
	cb := &v2.ComplianceBenchmark{}
	cb.SetName("CIS")
	cb.SetShortName("OCP_CIS")
	cb.SetVersion("1-5")
	cp := &v2.ComplianceProfile{}
	cp.SetId(ProfileUID)
	cp.SetName("ocp-cis")
	cp.SetProfileVersion("4.2")
	cp.SetDescription("this is a test")
	cp.SetRules(v2ApiRules)
	cp.SetTitle("Openshift CIS testing")
	cp.SetProductType("")
	cp.SetStandards([]*v2.ComplianceBenchmark{cb})
	cp.SetProduct("")
	cp.SetValues(values)
	return cp
}

// GetProfilesV2Api returns a list of v2 APIs that match GetProfilesV2Storage
func GetProfilesV2Api(_ *testing.T) []*v2.ComplianceProfile {
	return []*v2.ComplianceProfile{
		v2.ComplianceProfile_builder{
			Id:             ProfileUID,
			Name:           "ocp-cis",
			ProfileVersion: "4.2",
			Description:    "this is a test",
			Rules:          v2ApiRules,
			Title:          "Openshift CIS testing",
			ProductType:    "",
			Standards: []*v2.ComplianceBenchmark{v2.ComplianceBenchmark_builder{
				Name:      "CIS",
				ShortName: "OCP_CIS",
				Version:   "1-5",
			}.Build()},
			Product: "",
			Values:  values,
		}.Build(),
		v2.ComplianceProfile_builder{
			Id:             profileUID2,
			Name:           "rhcos-moderate",
			ProfileVersion: "4.1.2",
			Description:    "this is a test",
			Rules:          v2ApiRules,
			Title:          "Openshift CIS testing",
			ProductType:    "",
			Standards: []*v2.ComplianceBenchmark{v2.ComplianceBenchmark_builder{
				Name:      "CIS",
				ShortName: "OCP_CIS",
				Version:   "1-5",
			}.Build()},
			Product: "",
			Values:  values,
		}.Build(),
	}
}

// GetComplianceRemediationV2Msg -- returns a V2 message from sensor
func GetComplianceRemediationV2Msg(_ *testing.T) *central.ComplianceOperatorRemediationV2 {
	corv2 := &central.ComplianceOperatorRemediationV2{}
	corv2.SetId(RemediationUID)
	corv2.SetName("ocp4-stig-project-config-and-template-network-policy-1")
	corv2.SetComplianceCheckResultName("ocp4-stig-project-config-and-template-network-policy")
	corv2.SetApply(true)
	corv2.SetCurrentObject("")
	corv2.SetOutdatedObject("")
	corv2.SetEnforcementType("test-type")
	return corv2
}

// GetComplianceRemediationV2Storage -- returns a v2 message from sensor
func GetComplianceRemediationV2Storage(_ *testing.T) *storage.ComplianceOperatorRemediationV2 {
	corv2 := &storage.ComplianceOperatorRemediationV2{}
	corv2.SetId(RemediationUID)
	corv2.SetName("ocp4-stig-project-config-and-template-network-policy-1")
	corv2.SetComplianceCheckResultName("ocp4-stig-project-config-and-template-network-policy")
	corv2.SetApply(true)
	corv2.SetCurrentObject("")
	corv2.SetOutdatedObject("")
	corv2.SetEnforcementType("test-type")
	corv2.SetClusterId(fixtureconsts.Cluster1)
	return corv2
}
