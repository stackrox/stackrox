package validation

import (
	"testing"

	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/require"
)

func TestValidateScheduleRejectsInvalidBoundsAndIntervals(t *testing.T) {
	tests := map[string]*apiV2.ReportSchedule{
		"negative hour": {IntervalType: apiV2.ReportSchedule_DAILY, Hour: -1},
		"hour 24":       {IntervalType: apiV2.ReportSchedule_DAILY, Hour: 24},
		"negative minute": {IntervalType: apiV2.ReportSchedule_DAILY, Minute: -1},
		"minute 60":       {IntervalType: apiV2.ReportSchedule_DAILY, Minute: 60},
		"unknown interval": {
			IntervalType: apiV2.ReportSchedule_IntervalType(99),
		},
	}

	for name, schedule := range tests {
		t.Run(name, func(t *testing.T) {
			err := (&Validator{}).validateSchedule(&apiV2.ReportConfiguration{Schedule: schedule})
			require.ErrorIs(t, err, errox.InvalidArgs)
		})
	}
}
