package testutils

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	// ProfileUID -- profile UID used in test objects
	ProfileUID = uuid.NewV4().String()

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
)

// GetProfileV1SensorMsg -- returns a V1 storage object
func GetProfileV1SensorMsg(_ *testing.T) *storage.ComplianceOperatorProfile {
	return &storage.ComplianceOperatorProfile{
		Id:          ProfileUID,
		ProfileId:   profileID,
		Name:        "ocp-cis",
		ClusterId:   fixtureconsts.Cluster1,
		Description: "this is a test",
		Labels:      nil,
		Annotations: nil,
		Rules:       v1StorageRules,
	}
}

// GetProfileV2SensorMsg -- returns a V2 message from sensor
func GetProfileV2SensorMsg(_ *testing.T) *central.ComplianceOperatorProfileV2 {
	return &central.ComplianceOperatorProfileV2{
		Id:             ProfileUID,
		ProfileId:      profileID,
		Name:           "ocp-cis",
		ProfileVersion: "4.2",
		Description:    "this is a test",
		Labels:         nil,
		Annotations:    nil,
		Rules:          v2SensorRules,
		Title:          "Openshift CIS testing",
		Values:         values,
	}
}

// GetProfileV2Storage -- returns a V2 storage object
func GetProfileV2Storage(_ *testing.T) *storage.ComplianceOperatorProfileV2 {
	return &storage.ComplianceOperatorProfileV2{
		Id:             "ocp-cis-4.2",
		ProfileId:      profileID,
		Name:           "ocp-cis",
		ProfileVersion: "4.2",
		Description:    "this is a test",
		Labels:         nil,
		Annotations:    nil,
		Rules:          v2StorageRules,
		Title:          "Openshift CIS testing",
		ProductType:    "",
		Standard:       "",
		Product:        "",
		Values:         values,
	}
}
