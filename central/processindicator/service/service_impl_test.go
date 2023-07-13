package service

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/processbaseline/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestIndicatorsToGroupedResponses(t *testing.T) {
	var cases = []struct {
		name                string
		indicators          []*storage.ProcessIndicator
		nameGroups          []*v1.ProcessNameGroup
		nameContainerGroups []*v1.ProcessNameAndContainerNameGroup
	}{
		{
			name: "test grouping",
			indicators: []*storage.ProcessIndicator{
				{
					ContainerName: "one",
					Signal: &storage.ProcessSignal{
						Id:           "1",
						ExecFilePath: "cat",
						Args:         "hello",
						ContainerId:  "A",
					},
				},
				{
					ContainerName: "one",
					Signal: &storage.ProcessSignal{
						Id:           "2",
						ExecFilePath: "cat",
						Args:         "hello",
						ContainerId:  "B",
					},
				},
				{
					ContainerName: "one",
					Signal: &storage.ProcessSignal{
						Id:           "3",
						ExecFilePath: "cat",
						Args:         "boo",
						ContainerId:  "A",
					},
				},
				{
					ContainerName: "one",
					Signal: &storage.ProcessSignal{
						Id:           "4",
						ExecFilePath: "blah",
						Args:         "boo",
						ContainerId:  "C",
					},
				},
				{
					ContainerName: "two",
					Signal: &storage.ProcessSignal{
						Id:           "5",
						ExecFilePath: "grah",
						Args:         "boo",
						ContainerId:  "D",
					},
				},
			},
			nameGroups: []*v1.ProcessNameGroup{
				{
					Name:          "blah",
					TimesExecuted: 1,
					Groups: []*v1.ProcessGroup{
						{
							Args: "boo",
							Signals: []*storage.ProcessIndicator{
								{
									ContainerName: "one",
									Signal: &storage.ProcessSignal{
										Id:           "4",
										ExecFilePath: "blah",
										Args:         "boo",
										ContainerId:  "C",
									},
								},
							},
						},
					},
				},
				{
					Name:          "cat",
					TimesExecuted: 2,
					Groups: []*v1.ProcessGroup{
						{
							Args: "boo",
							Signals: []*storage.ProcessIndicator{
								{
									ContainerName: "one",
									Signal: &storage.ProcessSignal{
										Id:           "3",
										ExecFilePath: "cat",
										Args:         "boo",
										ContainerId:  "A",
									},
								},
							},
						},
						{
							Args: "hello",
							Signals: []*storage.ProcessIndicator{
								{
									ContainerName: "one",
									Signal: &storage.ProcessSignal{
										Id:           "1",
										ExecFilePath: "cat",
										Args:         "hello",
										ContainerId:  "A",
									},
								},
								{
									ContainerName: "one",
									Signal: &storage.ProcessSignal{
										Id:           "2",
										ExecFilePath: "cat",
										Args:         "hello",
										ContainerId:  "B",
									},
								},
							},
						},
					},
				},
				{
					Name:          "grah",
					TimesExecuted: 1,
					Groups: []*v1.ProcessGroup{
						{
							Args: "boo",
							Signals: []*storage.ProcessIndicator{
								{
									ContainerName: "two",
									Signal: &storage.ProcessSignal{
										Id:           "5",
										ExecFilePath: "grah",
										Args:         "boo",
										ContainerId:  "D",
									},
								},
							},
						},
					},
				},
			},
			nameContainerGroups: []*v1.ProcessNameAndContainerNameGroup{
				{
					Name:          "blah",
					ContainerName: "one",
					TimesExecuted: 1,
					Groups: []*v1.ProcessGroup{
						{
							Args: "boo",
							Signals: []*storage.ProcessIndicator{
								{
									ContainerName: "one",
									Signal: &storage.ProcessSignal{
										Id:           "4",
										ExecFilePath: "blah",
										Args:         "boo",
										ContainerId:  "C",
									},
								},
							},
						},
					},
				},
				{
					Name:          "cat",
					ContainerName: "one",
					TimesExecuted: 2,
					Groups: []*v1.ProcessGroup{
						{
							Args: "boo",
							Signals: []*storage.ProcessIndicator{
								{
									ContainerName: "one",
									Signal: &storage.ProcessSignal{
										Id:           "3",
										ExecFilePath: "cat",
										Args:         "boo",
										ContainerId:  "A",
									},
								},
							},
						},
						{
							Args: "hello",
							Signals: []*storage.ProcessIndicator{
								{
									ContainerName: "one",
									Signal: &storage.ProcessSignal{
										Id:           "1",
										ExecFilePath: "cat",
										Args:         "hello",
										ContainerId:  "A",
									},
								},
								{
									ContainerName: "one",
									Signal: &storage.ProcessSignal{
										Id:           "2",
										ExecFilePath: "cat",
										Args:         "hello",
										ContainerId:  "B",
									},
								},
							},
						},
					},
				},
				{
					Name:          "grah",
					ContainerName: "two",
					TimesExecuted: 1,
					Groups: []*v1.ProcessGroup{
						{
							Args: "boo",
							Signals: []*storage.ProcessIndicator{
								{
									ContainerName: "two",
									Signal: &storage.ProcessSignal{
										Id:           "5",
										ExecFilePath: "grah",
										Args:         "boo",
										ContainerId:  "D",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			testResults := IndicatorsToGroupedResponses(c.indicators)
			assert.Equal(t, c.nameGroups, testResults)
			testResultsWithContainer := indicatorsToGroupedResponsesWithContainer(c.indicators)
			assert.Equal(t, c.nameContainerGroups, testResultsWithContainer)
		})
	}
}

func TestBaselineCheck(t *testing.T) {
	var cases = []struct {
		name               string
		nameContainerGroup *v1.ProcessNameAndContainerNameGroup
		baselineElements   []*storage.BaselineElement
		baselineExists     bool
		baselineLocked     bool
		expectedSuspicious bool
	}{
		{
			name: "In the baseline",
			nameContainerGroup: &v1.ProcessNameAndContainerNameGroup{
				Name:          "Name One",
				ContainerName: "Container One",
			},
			baselineElements:   []*storage.BaselineElement{fixtures.GetBaselineElement("Name One")},
			baselineExists:     true,
			baselineLocked:     true,
			expectedSuspicious: false,
		},
		{
			name: "Not in the baseline",
			nameContainerGroup: &v1.ProcessNameAndContainerNameGroup{
				Name:          "Name Two",
				ContainerName: "Container One",
			},
			baselineElements:   []*storage.BaselineElement{fixtures.GetBaselineElement("Name One")},
			baselineExists:     true,
			baselineLocked:     true,
			expectedSuspicious: true,
		},
		{
			name: "No baseline",
			nameContainerGroup: &v1.ProcessNameAndContainerNameGroup{
				Name:          "Name One",
				ContainerName: "Container One",
			},
			baselineExists:     false,
			expectedSuspicious: false,
		},
		{
			name: "Unlocked baseline",
			nameContainerGroup: &v1.ProcessNameAndContainerNameGroup{
				Name:          "Name Two",
				ContainerName: "Container One",
			},
			baselineElements:   []*storage.BaselineElement{fixtures.GetBaselineElement("Name One")},
			baselineExists:     true,
			baselineLocked:     false,
			expectedSuspicious: false,
		},
		{
			name: "Empty locked baseline",
			nameContainerGroup: &v1.ProcessNameAndContainerNameGroup{
				Name:          "Name One",
				ContainerName: "Container One",
			},
			baselineElements:   []*storage.BaselineElement{},
			baselineExists:     true,
			baselineLocked:     true,
			expectedSuspicious: true,
		},
		{
			name: "Empty unlocked baseline",
			nameContainerGroup: &v1.ProcessNameAndContainerNameGroup{
				Name:          "Name One",
				ContainerName: "Container One",
			},
			baselineElements:   []*storage.BaselineElement{},
			baselineExists:     true,
			baselineLocked:     false,
			expectedSuspicious: false,
		},
	}
	hasReadCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension, resources.Deployment)))
	mockCtrl := gomock.NewController(t)
	deployments := deploymentMocks.NewMockDataStore(mockCtrl)
	baselines := mocks.NewMockDataStore(mockCtrl)
	service := serviceImpl{deployments: deployments, baselines: baselines}
	testClusterID := "Test"
	testNamespace := "Test"
	testDeploymentID := "Test"
	testStart := types.TimestampNow()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var baseline *storage.ProcessBaseline
			if c.baselineExists {
				baseline = fixtures.GetProcessBaseline()
				baseline.Elements = c.baselineElements
				if c.baselineLocked {
					baseline.UserLockedTimestamp = testStart
				}
			}
			deployment := &storage.Deployment{
				ClusterId: testClusterID,
				Namespace: testNamespace,
				Id:        testDeploymentID,
			}
			key := &storage.ProcessBaselineKey{
				ClusterId:     testClusterID,
				Namespace:     testNamespace,
				DeploymentId:  testDeploymentID,
				ContainerName: c.nameContainerGroup.ContainerName,
			}
			deployments.EXPECT().GetDeployment(gomock.Any(), testDeploymentID).Return(deployment, true, nil)
			baselines.EXPECT().GetProcessBaseline(gomock.Any(), key).Return(baseline, true, nil)
			err := service.setSuspicious(hasReadCtx, []*v1.ProcessNameAndContainerNameGroup{c.nameContainerGroup}, testDeploymentID)
			assert.NoError(t, err)
			assert.Equal(t, c.expectedSuspicious, c.nameContainerGroup.Suspicious)
		})
	}
}
