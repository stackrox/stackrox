package evaluator

import (
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	processIndicatorMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	processWhitelistMocks "github.com/stackrox/rox/central/processwhitelist/datastore/mocks"
	processWhitelistResultMocks "github.com/stackrox/rox/central/processwhitelistresults/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeWhitelistStatuses(t *testing.T, statuses ...string) (protoStatuses []storage.ContainerNameAndWhitelistStatus_WhitelistStatus) {
	for _, status := range statuses {
		protoStatusInt, ok := storage.ContainerNameAndWhitelistStatus_WhitelistStatus_value[strings.ToUpper(status)]
		require.True(t, ok, "invalid status %s", status)
		protoStatuses = append(protoStatuses, storage.ContainerNameAndWhitelistStatus_WhitelistStatus(protoStatusInt))
	}
	return
}

func TestProcessWhitelistEvaluator(t *testing.T) {
	deployment := fixtures.GetDeployment()

	cases := []struct {
		name         string
		whitelist    *storage.ProcessWhitelist
		whitelistErr error
		indicators   []*storage.ProcessIndicator
		indicatorErr error
		// Specify expectedIndicators as indices into the indicators slice above.
		expectedIndicatorIndices []int

		whitelistStatuses          []storage.ContainerNameAndWhitelistStatus_WhitelistStatus
		anomalousProcessesExecuted []bool
	}{
		{
			name:                       "No Whitelist",
			whitelistStatuses:          makeWhitelistStatuses(t, "NOT_GENERATED", "NOT_GENERATED"),
			anomalousProcessesExecuted: []bool{false, false},
		},
		{
			name:      "Whitelist exists, but not locked",
			whitelist: &storage.ProcessWhitelist{},
			indicators: []*storage.ProcessIndicator{
				{
					Signal: &storage.ProcessSignal{
						Name: "apt-get",
						Args: "install nmap",
					},
					ContainerName: deployment.GetContainers()[0].GetName(),
				},
			},
			whitelistStatuses:          makeWhitelistStatuses(t, "UNLOCKED", "UNLOCKED"),
			anomalousProcessesExecuted: []bool{false, false},
		},
		{
			name: "Locked whitelist, but all whitelisted",
			whitelist: &storage.ProcessWhitelist{
				StackRoxLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
				Elements:                fixtures.MakeWhitelistElements("/bin/apt-get", "/unrelated"),
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
			whitelistStatuses:          makeWhitelistStatuses(t, "LOCKED", "LOCKED"),
			anomalousProcessesExecuted: []bool{false, false},
		},
		{
			name: "Locked whitelist, one non-whitelisted process",
			whitelist: &storage.ProcessWhitelist{
				StackRoxLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
			},
			indicators: []*storage.ProcessIndicator{
				{
					Signal: &storage.ProcessSignal{
						Name: "apt-get",
						Args: "install nmap",
					},
					ContainerName: deployment.GetContainers()[0].GetName(),
				},
			},
			expectedIndicatorIndices:   []int{0},
			whitelistStatuses:          makeWhitelistStatuses(t, "LOCKED", "LOCKED"),
			anomalousProcessesExecuted: []bool{true, false},
		},
		{
			name: "Locked whitelist, two non-whitelisted processes",
			whitelist: &storage.ProcessWhitelist{
				StackRoxLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
			},
			indicators: []*storage.ProcessIndicator{
				{
					Signal: &storage.ProcessSignal{
						Name: "apt-get",
						Args: "install nmap",
					},
					ContainerName: deployment.GetContainers()[1].GetName(),
				},
				{
					Signal: &storage.ProcessSignal{
						Name: "curl",
						Args: "badssl.com",
					},
					ContainerName: deployment.GetContainers()[1].GetName(),
				},
			},
			expectedIndicatorIndices:   []int{0, 1},
			whitelistStatuses:          makeWhitelistStatuses(t, "LOCKED", "LOCKED"),
			anomalousProcessesExecuted: []bool{false, true},
		},
		{
			name: "Locked whitelist, two non-whitelisted processes from different containers",
			whitelist: &storage.ProcessWhitelist{
				StackRoxLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
				Elements:                fixtures.MakeWhitelistElements("/bin/apt-get"),
			},
			indicators: []*storage.ProcessIndicator{
				{
					Signal: &storage.ProcessSignal{
						Name:         "not-apt-get",
						ExecFilePath: "/bin/not-apt-get",
						Args:         "install nmap",
					},
					ContainerName: deployment.GetContainers()[0].GetName(),
				},
				{
					Signal: &storage.ProcessSignal{
						Name:         "apt-get",
						ExecFilePath: "/bin/apt-get",
						Args:         "install nmap",
					},
					ContainerName: deployment.GetContainers()[0].GetName(),
				},
				{
					Signal: &storage.ProcessSignal{
						Name:         "curl",
						ExecFilePath: "/bin/curl",
						Args:         "badssl.com",
					},
					ContainerName: deployment.GetContainers()[1].GetName(),
				},
			},
			expectedIndicatorIndices:   []int{0, 2},
			whitelistStatuses:          makeWhitelistStatuses(t, "LOCKED", "LOCKED"),
			anomalousProcessesExecuted: []bool{true, true},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockWhitelists := processWhitelistMocks.NewMockDataStore(mockCtrl)
			mockIndicators := processIndicatorMocks.NewMockDataStore(mockCtrl)
			mockResults := processWhitelistResultMocks.NewMockDataStore(mockCtrl)

			mockWhitelists.EXPECT().GetProcessWhitelist(gomock.Any(), gomock.Any()).MaxTimes(len(deployment.GetContainers())).Return(c.whitelist, c.whitelistErr)
			mockIndicators.EXPECT().SearchRawProcessIndicators(gomock.Any(), gomock.Any()).Return(c.indicators, c.indicatorErr)

			expectedWhitelistResult := &storage.ProcessWhitelistResults{
				DeploymentId: deployment.GetId(),
				ClusterId:    deployment.GetClusterId(),
				Namespace:    deployment.GetNamespace(),
			}
			for i, container := range deployment.GetContainers() {
				expectedWhitelistResult.WhitelistStatuses = append(expectedWhitelistResult.WhitelistStatuses, &storage.ContainerNameAndWhitelistStatus{
					ContainerName:              container.GetName(),
					WhitelistStatus:            c.whitelistStatuses[i],
					AnomalousProcessesExecuted: c.anomalousProcessesExecuted[i],
				})
			}
			mockResults.EXPECT().UpsertWhitelistResults(gomock.Any(), expectedWhitelistResult).Return(nil)

			results, err := New(mockResults, mockWhitelists, mockIndicators).EvaluateWhitelistsAndPersistResult(deployment)
			require.NoError(t, err)

			expectedIndicators := make([]*storage.ProcessIndicator, 0, len(c.expectedIndicatorIndices))
			for _, idx := range c.expectedIndicatorIndices {
				expectedIndicators = append(expectedIndicators, c.indicators[idx])
			}
			assert.ElementsMatch(t, results, expectedIndicators)
		})
	}
}
