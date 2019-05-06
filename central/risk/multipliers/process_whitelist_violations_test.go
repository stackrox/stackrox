package multipliers

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	processIndicatorMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	processWhitelistMocks "github.com/stackrox/rox/central/processwhitelist/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/assert"
)

func TestProcessWhitelists(t *testing.T) {
	deployment := getMockDeployment()
	cases := []struct {
		name         string
		whitelist    *storage.ProcessWhitelist
		whitelistErr error
		indicators   []*storage.ProcessIndicator
		indicatorErr error
		expected     *storage.Risk_Result
	}{
		{
			name: "No Whitelist",
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
		},
		{
			name: "Locked whitelist, but all whitelisted",
			whitelist: &storage.ProcessWhitelist{
				StackRoxLockedTimestamp: protoconv.MustConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
				Elements:                fixtures.MakeWhitelistElements("apt-get", "unrelated"),
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
			expected: &storage.Risk_Result{
				Name:  processWhitelistHeading,
				Score: 1.6,
				Factors: []*storage.Risk_Result_Factor{
					{Message: "Detected execution of suspicious process \"apt-get\" with args \"install nmap\" in container containerName"},
				},
			},
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
					ContainerName: deployment.GetContainers()[0].GetName(),
				},
				{
					Signal: &storage.ProcessSignal{
						Name: "curl",
						Args: "badssl.com",
					},
					ContainerName: deployment.GetContainers()[0].GetName(),
				},
			},
			expected: &storage.Risk_Result{
				Name:  processWhitelistHeading,
				Score: 2.14,
				Factors: []*storage.Risk_Result_Factor{
					{Message: "Detected execution of suspicious process \"apt-get\" with args \"install nmap\" in container containerName"},
					{Message: "Detected execution of suspicious process \"curl\" with args \"badssl.com\" in container containerName"},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockWhitelists := processWhitelistMocks.NewMockDataStore(mockCtrl)
			mockIndicators := processIndicatorMocks.NewMockDataStore(mockCtrl)

			mockWhitelists.EXPECT().GetProcessWhitelist(gomock.Any()).MaxTimes(len(deployment.GetContainers())).Return(c.whitelist, c.whitelistErr)
			mockIndicators.EXPECT().SearchRawProcessIndicators(gomock.Any()).Return(c.indicators, c.indicatorErr)

			result := NewProcessWhitelists(mockWhitelists, mockIndicators).Score(deployment)
			assert.ElementsMatch(t, c.expected.GetFactors(), result.GetFactors())
			assert.InDelta(t, c.expected.GetScore(), result.GetScore(), 0.001)
		})
	}
}
