package evaluator

import (
	"strings"
	"testing"
	"time"

	processBaselineMocks "github.com/stackrox/rox/central/processbaseline/datastore/mocks"
	processBaselineResultMocks "github.com/stackrox/rox/central/processbaselineresults/datastore/mocks"
	processIndicatorMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func makeBaselineStatuses(t *testing.T, statuses ...string) (protoStatuses []storage.ContainerNameAndBaselineStatus_BaselineStatus) {
	for _, status := range statuses {
		protoStatusInt, ok := storage.ContainerNameAndBaselineStatus_BaselineStatus_value[strings.ToUpper(status)]
		require.True(t, ok, "invalid status %s", status)
		protoStatuses = append(protoStatuses, storage.ContainerNameAndBaselineStatus_BaselineStatus(protoStatusInt))
	}
	return
}

func TestProcessBaselineEvaluator(t *testing.T) {
	deployment := fixtures.GetDeployment()

	cases := []struct {
		name         string
		baseline     *storage.ProcessBaseline
		baselineErr  error
		indicators   []*storage.ProcessIndicator
		indicatorErr error
		// Specify expectedIndicators as indices into the indicators slice above.
		expectedIndicatorIndices []int

		baselineStatuses           []storage.ContainerNameAndBaselineStatus_BaselineStatus
		anomalousProcessesExecuted []bool

		currentBaselineResults *storage.ProcessBaselineResults
		shouldBePersisted      bool
	}{
		{
			name:                       "No Process Baseline",
			baselineStatuses:           makeBaselineStatuses(t, "NOT_GENERATED", "NOT_GENERATED"),
			anomalousProcessesExecuted: []bool{false, false},
			currentBaselineResults:     nil,
			shouldBePersisted:          true,
		},
		{
			name:                       "Process Baseline exists, but not locked",
			baseline:                   &storage.ProcessBaseline{},
			baselineStatuses:           makeBaselineStatuses(t, "UNLOCKED", "UNLOCKED"),
			anomalousProcessesExecuted: []bool{false, false},
			currentBaselineResults:     nil,
			shouldBePersisted:          true,
		},
		{
			name: "Locked process baseline, but all processes in baseline",
			baseline: &storage.ProcessBaseline{
				StackRoxLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
				Elements:                fixtures.MakeBaselineElements("/bin/apt-get", "/unrelated"),
			},
			indicators: []*storage.ProcessIndicator{
				{
					Signal: &storage.ProcessSignal{
						ExecFilePath: "/bin/apt-get",
						Args:         "install nmap",
					},
					ContainerName: deployment.GetContainers()[0].GetName(),
				},
			},
			baselineStatuses:           makeBaselineStatuses(t, "LOCKED", "LOCKED"),
			anomalousProcessesExecuted: []bool{false, false},
			currentBaselineResults:     nil,
			shouldBePersisted:          true,
		},
		{
			name: "Locked process baseline, one not-in-baseline process",
			baseline: &storage.ProcessBaseline{
				StackRoxLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
			},
			indicators: []*storage.ProcessIndicator{
				{
					Signal: &storage.ProcessSignal{
						ExecFilePath: "apt-get",
						Args:         "install nmap",
					},
					ContainerName: deployment.GetContainers()[0].GetName(),
				},
			},
			expectedIndicatorIndices:   []int{0},
			baselineStatuses:           makeBaselineStatuses(t, "LOCKED", "LOCKED"),
			anomalousProcessesExecuted: []bool{true, false},
			currentBaselineResults:     nil,
			shouldBePersisted:          true,
		},
		{
			name: "Locked process baseline, two not-in-baseline processes",
			baseline: &storage.ProcessBaseline{
				StackRoxLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
			},
			indicators: []*storage.ProcessIndicator{
				{
					Signal: &storage.ProcessSignal{
						ExecFilePath: "apt-get",
						Args:         "install nmap",
					},
					ContainerName: deployment.GetContainers()[1].GetName(),
				},
				{
					Signal: &storage.ProcessSignal{
						ExecFilePath: "curl",
						Args:         "badssl.com",
					},
					ContainerName: deployment.GetContainers()[1].GetName(),
				},
			},
			expectedIndicatorIndices:   []int{0, 1},
			baselineStatuses:           makeBaselineStatuses(t, "LOCKED", "LOCKED"),
			anomalousProcessesExecuted: []bool{false, true},
			currentBaselineResults:     nil,
			shouldBePersisted:          true,
		},
		{
			name: "Locked process baseline, two not-in-baseline processes from different containers",
			baseline: &storage.ProcessBaseline{
				StackRoxLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
				Elements:                fixtures.MakeBaselineElements("/bin/apt-get"),
			},
			indicators: []*storage.ProcessIndicator{
				{
					Signal: &storage.ProcessSignal{
						ExecFilePath: "/bin/not-apt-get",
						Args:         "install nmap",
					},
					ContainerName: deployment.GetContainers()[0].GetName(),
				},
				{
					Signal: &storage.ProcessSignal{
						ExecFilePath: "/bin/apt-get",
						Args:         "install nmap",
					},
					ContainerName: deployment.GetContainers()[0].GetName(),
				},
				{
					Signal: &storage.ProcessSignal{
						ExecFilePath: "/bin/curl",
						Args:         "badssl.com",
					},
					ContainerName: deployment.GetContainers()[1].GetName(),
				},
			},
			expectedIndicatorIndices:   []int{0, 2},
			baselineStatuses:           makeBaselineStatuses(t, "LOCKED", "LOCKED"),
			anomalousProcessesExecuted: []bool{true, true},
			currentBaselineResults:     nil,
			shouldBePersisted:          true,
		},
		{
			name: "Locked process baseline, two not-in-baseline processes from different containers. result already exists",
			baseline: &storage.ProcessBaseline{
				StackRoxLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
				Elements:                fixtures.MakeBaselineElements("/bin/apt-get"),
			},
			indicators: []*storage.ProcessIndicator{
				{
					Signal: &storage.ProcessSignal{
						ExecFilePath: "/bin/not-apt-get",
						Args:         "install nmap",
					},
					ContainerName: deployment.GetContainers()[0].GetName(),
				},
				{
					Signal: &storage.ProcessSignal{
						ExecFilePath: "/bin/apt-get",
						Args:         "install nmap",
					},
					ContainerName: deployment.GetContainers()[0].GetName(),
				},
				{
					Signal: &storage.ProcessSignal{
						ExecFilePath: "/bin/curl",
						Args:         "badssl.com",
					},
					ContainerName: deployment.GetContainers()[1].GetName(),
				},
			},
			expectedIndicatorIndices:   []int{0, 2},
			baselineStatuses:           makeBaselineStatuses(t, "LOCKED", "LOCKED"),
			anomalousProcessesExecuted: []bool{true, true},
			currentBaselineResults: &storage.ProcessBaselineResults{
				BaselineStatuses: []*storage.ContainerNameAndBaselineStatus{
					{
						ContainerName:              deployment.GetContainers()[1].GetName(),
						BaselineStatus:             storage.ContainerNameAndBaselineStatus_LOCKED,
						AnomalousProcessesExecuted: true,
					},
					{
						ContainerName:              deployment.GetContainers()[0].GetName(),
						BaselineStatus:             storage.ContainerNameAndBaselineStatus_LOCKED,
						AnomalousProcessesExecuted: true,
					},
				},
			},
			shouldBePersisted: false,
		},
		{
			name: "Locked process baseline, two not-in-baseline processes from different containers. result already exists, but needs an update",
			baseline: &storage.ProcessBaseline{
				StackRoxLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
				Elements:                fixtures.MakeBaselineElements("/bin/apt-get"),
			},
			indicators: []*storage.ProcessIndicator{
				{
					Signal: &storage.ProcessSignal{
						ExecFilePath: "/bin/not-apt-get",
						Args:         "install nmap",
					},
					ContainerName: deployment.GetContainers()[0].GetName(),
				},
				{
					Signal: &storage.ProcessSignal{
						ExecFilePath: "/bin/apt-get",
						Args:         "install nmap",
					},
					ContainerName: deployment.GetContainers()[0].GetName(),
				},
				{
					Signal: &storage.ProcessSignal{
						ExecFilePath: "/bin/curl",
						Args:         "badssl.com",
					},
					ContainerName: deployment.GetContainers()[1].GetName(),
				},
			},
			expectedIndicatorIndices:   []int{0, 2},
			baselineStatuses:           makeBaselineStatuses(t, "LOCKED", "LOCKED"),
			anomalousProcessesExecuted: []bool{true, true},
			currentBaselineResults: &storage.ProcessBaselineResults{
				BaselineStatuses: []*storage.ContainerNameAndBaselineStatus{
					{
						ContainerName:              deployment.GetContainers()[1].GetName(),
						BaselineStatus:             storage.ContainerNameAndBaselineStatus_UNLOCKED,
						AnomalousProcessesExecuted: true,
					},
					{
						ContainerName:              deployment.GetContainers()[0].GetName(),
						BaselineStatus:             storage.ContainerNameAndBaselineStatus_LOCKED,
						AnomalousProcessesExecuted: true,
					},
				},
			},
			shouldBePersisted: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockBaselines := processBaselineMocks.NewMockDataStore(mockCtrl)
			mockIndicators := processIndicatorMocks.NewMockDataStore(mockCtrl)
			mockResults := processBaselineResultMocks.NewMockDataStore(mockCtrl)

			mockBaselines.EXPECT().GetProcessBaseline(gomock.Any(), gomock.Any()).MaxTimes(len(deployment.GetContainers())).Return(c.baseline, c.baseline != nil, c.baselineErr)
			if c.indicators != nil {
				mockIndicators.EXPECT().SearchRawProcessIndicators(gomock.Any(), gomock.Any()).Return(c.indicators, c.indicatorErr)
			}

			expectedBaselineResult := &storage.ProcessBaselineResults{
				DeploymentId: deployment.GetId(),
				ClusterId:    deployment.GetClusterId(),
				Namespace:    deployment.GetNamespace(),
			}
			for i, container := range deployment.GetContainers() {
				expectedBaselineResult.BaselineStatuses = append(expectedBaselineResult.BaselineStatuses, &storage.ContainerNameAndBaselineStatus{
					ContainerName:              container.GetName(),
					BaselineStatus:             c.baselineStatuses[i],
					AnomalousProcessesExecuted: c.anomalousProcessesExecuted[i],
				})
			}
			mockResults.EXPECT().GetBaselineResults(gomock.Any(), deployment.GetId()).Return(c.currentBaselineResults, nil)

			if c.shouldBePersisted {
				mockResults.EXPECT().UpsertBaselineResults(gomock.Any(), expectedBaselineResult).Return(nil)
			}
			results, err := New(mockResults, mockBaselines, mockIndicators).EvaluateBaselinesAndPersistResult(deployment)
			require.NoError(t, err)

			expectedIndicators := make([]*storage.ProcessIndicator, 0, len(c.expectedIndicatorIndices))
			for _, idx := range c.expectedIndicatorIndices {
				expectedIndicators = append(expectedIndicators, c.indicators[idx])
			}
			assert.ElementsMatch(t, results, expectedIndicators)
		})
	}
}
