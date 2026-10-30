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
