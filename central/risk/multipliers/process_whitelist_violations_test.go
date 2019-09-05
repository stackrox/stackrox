package multipliers

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/processwhitelist/evaluator/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestProcessWhitelists(t *testing.T) {
	deployment := getMockDeployment()
	cases := []struct {
		name               string
		violatingProcesses []*storage.ProcessIndicator
		evaluatorErr       error
		expected           *storage.Risk_Result
	}{
		{
			name: "No violating processes",
		},
		{
			name: "Evaluator error",
			violatingProcesses: []*storage.ProcessIndicator{
				{
					Id: "SHOULD BE IGNORED",
				},
			},
			evaluatorErr: errors.New("here's an error"),
		},
		{
			name: "One violating process",
			violatingProcesses: []*storage.ProcessIndicator{
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
			name: "Two violating processes",
			violatingProcesses: []*storage.ProcessIndicator{
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

			mockEvaluator := mocks.NewMockEvaluator(mockCtrl)
			mockEvaluator.EXPECT().EvaluateWhitelistsAndPersistResult(deployment).Return(c.violatingProcesses, c.evaluatorErr)
			result := NewProcessWhitelists(mockEvaluator).Score(context.Background(), deployment, nil)
			assert.ElementsMatch(t, c.expected.GetFactors(), result.GetFactors())
			assert.InDelta(t, c.expected.GetScore(), result.GetScore(), 0.001)
		})
	}
}
