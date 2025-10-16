package resolvers

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestMapListAlertsToPolicyCounterResolver(t *testing.T) {
	alerts := []*storage.ListAlert{
		storage.ListAlert_builder{
			State: storage.ViolationState_ACTIVE,
			Policy: storage.ListAlertPolicy_builder{
				Id:       "id1",
				Severity: storage.Severity_LOW_SEVERITY,
			}.Build(),
		}.Build(),
		storage.ListAlert_builder{
			State: storage.ViolationState_ACTIVE,
			Policy: storage.ListAlertPolicy_builder{
				Id:       "id2",
				Severity: storage.Severity_HIGH_SEVERITY,
			}.Build(),
		}.Build(),
		storage.ListAlert_builder{
			State: storage.ViolationState_RESOLVED,
			Policy: storage.ListAlertPolicy_builder{
				Id:       "id3",
				Severity: storage.Severity_CRITICAL_SEVERITY,
			}.Build(),
		}.Build(),
		storage.ListAlert_builder{
			State: storage.ViolationState_ACTIVE,
			Policy: storage.ListAlertPolicy_builder{
				Id:       "id1",
				Severity: storage.Severity_LOW_SEVERITY,
			}.Build(),
		}.Build(),
		storage.ListAlert_builder{
			State: storage.ViolationState_ACTIVE,
			Policy: storage.ListAlertPolicy_builder{
				Id:       "id3",
				Severity: storage.Severity_LOW_SEVERITY,
			}.Build(),
		}.Build(),
	}

	counterResolver := mapListAlertsToPolicySeverityCount(alerts)
	assert.Equal(t, int32(3), counterResolver.total)
	assert.Equal(t, int32(2), counterResolver.low)
	assert.Equal(t, int32(0), counterResolver.medium)
	assert.Equal(t, int32(1), counterResolver.high)
	assert.Equal(t, int32(0), counterResolver.critical)
}
