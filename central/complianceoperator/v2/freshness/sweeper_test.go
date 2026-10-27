package freshness

import (
	"testing"
	"time"

	checkResultMocks "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/mocks"
	scanConfigMocks "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestExpectedRefresh(t *testing.T) {
	// Friday, 2024-03-15 10:30:00 UTC.
	now := time.Date(2024, time.March, 15, 10, 30, 0, 0, time.UTC)

	dailySchedule := &storage.Schedule{
		IntervalType: storage.Schedule_DAILY,
		Hour:         0,
		Minute:       0,
	}

	// Schedule with an unset interval type fails to convert to a crontab.
	invalidSchedule := &storage.Schedule{
		Hour:   0,
		Minute: 0,
	}

	cases := map[string]struct {
		sched       *storage.Schedule
		expectOK    bool
		expectedTod func(now time.Time) time.Time
	}{
		"nil schedule": {
			sched:    nil,
			expectOK: false,
		},
		"valid daily schedule": {
			sched:    dailySchedule,
			expectOK: true,
			expectedTod: func(now time.Time) time.Time {
				return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			},
		},
		"invalid schedule": {
			sched:    invalidSchedule,
			expectOK: false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			prev, ok := expectedRefresh(c.sched, now)
			assert.Equal(t, c.expectOK, ok)
			if !c.expectOK {
				return
			}
			require.False(t, prev.After(now), "expected prev tick to be at or before now")
			assert.True(t, prev.Equal(c.expectedTod(now)), "expected %v, got %v", c.expectedTod(now), prev)
		})
	}
}

func TestSweep(t *testing.T) {
	dailySchedule := &storage.Schedule{
		IntervalType: storage.Schedule_DAILY,
		Hour:         0,
		Minute:       0,
	}

	t.Run("updates freshness only for configs with a valid schedule", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockScanConfigDS := scanConfigMocks.NewMockDataStore(ctrl)
		mockCheckResultDS := checkResultMocks.NewMockDataStore(ctrl)

		scheduledConfig := &storage.ComplianceOperatorScanConfigurationV2{
			ScanConfigName: "scheduled-config",
			Schedule:       dailySchedule,
		}
		unscheduledConfig := &storage.ComplianceOperatorScanConfigurationV2{
			ScanConfigName: "unscheduled-config",
			Schedule:       nil,
		}

		mockScanConfigDS.EXPECT().
			GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return([]*storage.ComplianceOperatorScanConfigurationV2{scheduledConfig, unscheduledConfig}, nil)

		mockCheckResultDS.EXPECT().
			UpdateFreshnessByScanConfig(gomock.Any(), "scheduled-config", gomock.Any()).
			Times(1).
			Return(int64(3), nil)
		mockCheckResultDS.EXPECT().
			UpdateFreshnessByScanConfig(gomock.Any(), "unscheduled-config", gomock.Any()).
			Times(0)

		sweeper := New(mockScanConfigDS, mockCheckResultDS, time.Minute).(*sweeperImpl)
		sweeper.sweep()
	})

	t.Run("does not call UpdateFreshnessByScanConfig when listing configs fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockScanConfigDS := scanConfigMocks.NewMockDataStore(ctrl)
		mockCheckResultDS := checkResultMocks.NewMockDataStore(ctrl)

		mockScanConfigDS.EXPECT().
			GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return(nil, errox.NotFound)

		mockCheckResultDS.EXPECT().
			UpdateFreshnessByScanConfig(gomock.Any(), gomock.Any(), gomock.Any()).
			Times(0)

		sweeper := New(mockScanConfigDS, mockCheckResultDS, time.Minute).(*sweeperImpl)
		assert.NotPanics(t, func() { sweeper.sweep() })
	})

	t.Run("continues sweeping after an update failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockScanConfigDS := scanConfigMocks.NewMockDataStore(ctrl)
		mockCheckResultDS := checkResultMocks.NewMockDataStore(ctrl)

		firstConfig := &storage.ComplianceOperatorScanConfigurationV2{
			ScanConfigName: "first-config",
			Schedule:       dailySchedule,
		}
		secondConfig := &storage.ComplianceOperatorScanConfigurationV2{
			ScanConfigName: "second-config",
			Schedule:       dailySchedule,
		}

		mockScanConfigDS.EXPECT().
			GetScanConfigurations(gomock.Any(), gomock.Any()).
			Return([]*storage.ComplianceOperatorScanConfigurationV2{firstConfig, secondConfig}, nil)

		mockCheckResultDS.EXPECT().
			UpdateFreshnessByScanConfig(gomock.Any(), "first-config", gomock.Any()).
			Times(1).
			Return(int64(0), errox.InvariantViolation)
		mockCheckResultDS.EXPECT().
			UpdateFreshnessByScanConfig(gomock.Any(), "second-config", gomock.Any()).
			Times(1).
			Return(int64(1), nil)

		sweeper := New(mockScanConfigDS, mockCheckResultDS, time.Minute).(*sweeperImpl)
		assert.NotPanics(t, func() { sweeper.sweep() })
	})
}
