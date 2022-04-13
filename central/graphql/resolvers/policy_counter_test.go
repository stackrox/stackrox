package resolvers

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestMapListAlertsToPolicyCounterResolver(t *testing.T) {
	alerts := []*storage.ListAlert{
		{
			State: storage.ViolationState_ACTIVE,
			Policy: &storage.ListAlertPolicy{
				Id:       "id1",
				Severity: storage.Severity_LOW_SEVERITY,
			},
		},
		{
			State: storage.ViolationState_ACTIVE,
			Policy: &storage.ListAlertPolicy{
				Id:       "id2",
				Severity: storage.Severity_HIGH_SEVERITY,
			},
		},
		{
			State: storage.ViolationState_RESOLVED,
			Policy: &storage.ListAlertPolicy{
				Id:       "id3",
				Severity: storage.Severity_CRITICAL_SEVERITY,
			},
		},
		{
			State: storage.ViolationState_ACTIVE,
			Policy: &storage.ListAlertPolicy{
				Id:       "id1",
				Severity: storage.Severity_LOW_SEVERITY,
			},
		},
		{
			State: storage.ViolationState_ACTIVE,
			Policy: &storage.ListAlertPolicy{
				Id:       "id3",
				Severity: storage.Severity_LOW_SEVERITY,
			},
		},
	}

	counterResolver := mapListAlertsToPolicySeverityCount(alerts)
	assert.Equal(t, int32(3), counterResolver.total)
	assert.Equal(t, int32(2), counterResolver.low)
	assert.Equal(t, int32(0), counterResolver.medium)
	assert.Equal(t, int32(1), counterResolver.high)
	assert.Equal(t, int32(0), counterResolver.critical)
}
