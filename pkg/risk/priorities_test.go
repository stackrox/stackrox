package risk

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestPriorityClassificationSortOrder(t *testing.T) {
	assert.Equal(t, int32(1), PriorityClassificationSortOrder(storage.RiskPriorityClassification_RISK_PRIORITY_RESPOND_NOW))
	assert.Equal(t, int32(2), PriorityClassificationSortOrder(storage.RiskPriorityClassification_RISK_PRIORITY_INVESTIGATE_NOW))
	assert.Equal(t, int32(3), PriorityClassificationSortOrder(storage.RiskPriorityClassification_RISK_PRIORITY_REMEDIATE_NOW))
	assert.Equal(t, int32(4), PriorityClassificationSortOrder(storage.RiskPriorityClassification_RISK_PRIORITY_REMEDIATE))
	assert.Equal(t, int32(5), PriorityClassificationSortOrder(storage.RiskPriorityClassification_RISK_PRIORITY_IMPROVE_POSTURE))
	assert.Equal(t, int32(99), PriorityClassificationSortOrder(storage.RiskPriorityClassification_RISK_PRIORITY_UNKNOWN))
}

func TestActivityClassificationSortOrder(t *testing.T) {
	assert.Equal(t, int32(1), ActivityClassificationSortOrder(storage.RiskActivityClassification_RISK_ACTIVITY_RESPOND_NOW))
	assert.Equal(t, int32(2), ActivityClassificationSortOrder(storage.RiskActivityClassification_RISK_ACTIVITY_INVESTIGATE_NOW))
	assert.Equal(t, int32(3), ActivityClassificationSortOrder(storage.RiskActivityClassification_RISK_ACTIVITY_NONE_OBSERVED))
	assert.Equal(t, int32(99), ActivityClassificationSortOrder(storage.RiskActivityClassification_RISK_ACTIVITY_UNKNOWN))
}

func TestExposureClassificationSortOrder(t *testing.T) {
	assert.Equal(t, int32(1), ExposureClassificationSortOrder(storage.RiskExposureClassification_RISK_EXPOSURE_REMEDIATE_NOW))
	assert.Equal(t, int32(2), ExposureClassificationSortOrder(storage.RiskExposureClassification_RISK_EXPOSURE_REMEDIATE))
	assert.Equal(t, int32(3), ExposureClassificationSortOrder(storage.RiskExposureClassification_RISK_EXPOSURE_NONE_DETECTED))
	assert.Equal(t, int32(99), ExposureClassificationSortOrder(storage.RiskExposureClassification_RISK_EXPOSURE_UNKNOWN))
}

func TestPostureClassificationSortOrder(t *testing.T) {
	assert.Equal(t, int32(1), PostureClassificationSortOrder(storage.RiskPostureClassification_RISK_POSTURE_CRITICAL))
	assert.Equal(t, int32(2), PostureClassificationSortOrder(storage.RiskPostureClassification_RISK_POSTURE_HIGH))
	assert.Equal(t, int32(3), PostureClassificationSortOrder(storage.RiskPostureClassification_RISK_POSTURE_MEDIUM))
	assert.Equal(t, int32(4), PostureClassificationSortOrder(storage.RiskPostureClassification_RISK_POSTURE_LOW))
	assert.Equal(t, int32(5), PostureClassificationSortOrder(storage.RiskPostureClassification_RISK_POSTURE_NONE_DETECTED))
	assert.Equal(t, int32(99), PostureClassificationSortOrder(storage.RiskPostureClassification_RISK_POSTURE_UNKNOWN))
}
