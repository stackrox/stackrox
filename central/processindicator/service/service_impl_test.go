package service

import (
	"context"
	"testing"

	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/processbaseline/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
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
				storage.ProcessIndicator_builder{
					ContainerName: "one",
					Signal: storage.ProcessSignal_builder{
						Id:           "1",
						ExecFilePath: "cat",
						Args:         "hello",
						ContainerId:  "A",
					}.Build(),
				}.Build(),
				storage.ProcessIndicator_builder{
					ContainerName: "one",
					Signal: storage.ProcessSignal_builder{
						Id:           "2",
						ExecFilePath: "cat",
						Args:         "hello",
						ContainerId:  "B",
					}.Build(),
				}.Build(),
				storage.ProcessIndicator_builder{
					ContainerName: "one",
					Signal: storage.ProcessSignal_builder{
						Id:           "3",
						ExecFilePath: "cat",
						Args:         "boo",
						ContainerId:  "A",
					}.Build(),
				}.Build(),
				storage.ProcessIndicator_builder{
					ContainerName: "one",
					Signal: storage.ProcessSignal_builder{
						Id:           "4",
						ExecFilePath: "blah",
						Args:         "boo",
						ContainerId:  "C",
					}.Build(),
				}.Build(),
				storage.ProcessIndicator_builder{
					ContainerName: "two",
					Signal: storage.ProcessSignal_builder{
						Id:           "5",
						ExecFilePath: "grah",
						Args:         "boo",
						ContainerId:  "D",
					}.Build(),
				}.Build(),
			},
			nameGroups: []*v1.ProcessNameGroup{
				v1.ProcessNameGroup_builder{
					Name:          "blah",
					TimesExecuted: 1,
					Groups: []*v1.ProcessGroup{
						v1.ProcessGroup_builder{
							Args: "boo",
							Signals: []*storage.ProcessIndicator{
								storage.ProcessIndicator_builder{
									ContainerName: "one",
									Signal: storage.ProcessSignal_builder{
										Id:           "4",
										ExecFilePath: "blah",
										Args:         "boo",
										ContainerId:  "C",
									}.Build(),
								}.Build(),
							},
						}.Build(),
					},
				}.Build(),
				v1.ProcessNameGroup_builder{
					Name:          "cat",
					TimesExecuted: 2,
					Groups: []*v1.ProcessGroup{
						v1.ProcessGroup_builder{
							Args: "boo",
							Signals: []*storage.ProcessIndicator{
								storage.ProcessIndicator_builder{
									ContainerName: "one",
									Signal: storage.ProcessSignal_builder{
										Id:           "3",
										ExecFilePath: "cat",
										Args:         "boo",
										ContainerId:  "A",
									}.Build(),
								}.Build(),
							},
						}.Build(),
						v1.ProcessGroup_builder{
							Args: "hello",
							Signals: []*storage.ProcessIndicator{
								storage.ProcessIndicator_builder{
									ContainerName: "one",
									Signal: storage.ProcessSignal_builder{
										Id:           "1",
										ExecFilePath: "cat",
										Args:         "hello",
										ContainerId:  "A",
									}.Build(),
								}.Build(),
								storage.ProcessIndicator_builder{
									ContainerName: "one",
									Signal: storage.ProcessSignal_builder{
										Id:           "2",
										ExecFilePath: "cat",
										Args:         "hello",
										ContainerId:  "B",
									}.Build(),
								}.Build(),
							},
						}.Build(),
					},
				}.Build(),
				v1.ProcessNameGroup_builder{
					Name:          "grah",
					TimesExecuted: 1,
					Groups: []*v1.ProcessGroup{
						v1.ProcessGroup_builder{
							Args: "boo",
							Signals: []*storage.ProcessIndicator{
								storage.ProcessIndicator_builder{
									ContainerName: "two",
									Signal: storage.ProcessSignal_builder{
										Id:           "5",
										ExecFilePath: "grah",
										Args:         "boo",
										ContainerId:  "D",
									}.Build(),
								}.Build(),
							},
						}.Build(),
					},
				}.Build(),
			},
			nameContainerGroups: []*v1.ProcessNameAndContainerNameGroup{
				v1.ProcessNameAndContainerNameGroup_builder{
					Name:          "blah",
					ContainerName: "one",
					TimesExecuted: 1,
					Groups: []*v1.ProcessGroup{
						v1.ProcessGroup_builder{
							Args: "boo",
							Signals: []*storage.ProcessIndicator{
								storage.ProcessIndicator_builder{
									ContainerName: "one",
									Signal: storage.ProcessSignal_builder{
										Id:           "4",
										ExecFilePath: "blah",
										Args:         "boo",
										ContainerId:  "C",
									}.Build(),
								}.Build(),
							},
						}.Build(),
					},
				}.Build(),
				v1.ProcessNameAndContainerNameGroup_builder{
					Name:          "cat",
					ContainerName: "one",
					TimesExecuted: 2,
					Groups: []*v1.ProcessGroup{
						v1.ProcessGroup_builder{
							Args: "boo",
							Signals: []*storage.ProcessIndicator{
								storage.ProcessIndicator_builder{
									ContainerName: "one",
									Signal: storage.ProcessSignal_builder{
										Id:           "3",
										ExecFilePath: "cat",
										Args:         "boo",
										ContainerId:  "A",
									}.Build(),
								}.Build(),
							},
						}.Build(),
						v1.ProcessGroup_builder{
							Args: "hello",
							Signals: []*storage.ProcessIndicator{
								storage.ProcessIndicator_builder{
									ContainerName: "one",
									Signal: storage.ProcessSignal_builder{
										Id:           "1",
										ExecFilePath: "cat",
										Args:         "hello",
										ContainerId:  "A",
									}.Build(),
								}.Build(),
								storage.ProcessIndicator_builder{
									ContainerName: "one",
									Signal: storage.ProcessSignal_builder{
										Id:           "2",
										ExecFilePath: "cat",
										Args:         "hello",
										ContainerId:  "B",
									}.Build(),
								}.Build(),
							},
						}.Build(),
					},
				}.Build(),
				v1.ProcessNameAndContainerNameGroup_builder{
					Name:          "grah",
					ContainerName: "two",
					TimesExecuted: 1,
					Groups: []*v1.ProcessGroup{
						v1.ProcessGroup_builder{
							Args: "boo",
							Signals: []*storage.ProcessIndicator{
								storage.ProcessIndicator_builder{
									ContainerName: "two",
									Signal: storage.ProcessSignal_builder{
										Id:           "5",
										ExecFilePath: "grah",
										Args:         "boo",
										ContainerId:  "D",
									}.Build(),
								}.Build(),
							},
						}.Build(),
					},
				}.Build(),
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			testResults := IndicatorsToGroupedResponses(c.indicators)
			protoassert.SlicesEqual(t, c.nameGroups, testResults)
			testResultsWithContainer := indicatorsToGroupedResponsesWithContainer(c.indicators)
			protoassert.SlicesEqual(t, c.nameContainerGroups, testResultsWithContainer)
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
			nameContainerGroup: v1.ProcessNameAndContainerNameGroup_builder{
				Name:          "Name One",
				ContainerName: "Container One",
			}.Build(),
			baselineElements:   []*storage.BaselineElement{fixtures.GetBaselineElement("Name One")},
			baselineExists:     true,
			baselineLocked:     true,
			expectedSuspicious: false,
		},
		{
			name: "Not in the baseline",
			nameContainerGroup: v1.ProcessNameAndContainerNameGroup_builder{
				Name:          "Name Two",
				ContainerName: "Container One",
			}.Build(),
			baselineElements:   []*storage.BaselineElement{fixtures.GetBaselineElement("Name One")},
			baselineExists:     true,
			baselineLocked:     true,
			expectedSuspicious: true,
		},
		{
			name: "No baseline",
			nameContainerGroup: v1.ProcessNameAndContainerNameGroup_builder{
				Name:          "Name One",
				ContainerName: "Container One",
			}.Build(),
			baselineExists:     false,
			expectedSuspicious: false,
		},
		{
			name: "Unlocked baseline",
			nameContainerGroup: v1.ProcessNameAndContainerNameGroup_builder{
				Name:          "Name Two",
				ContainerName: "Container One",
			}.Build(),
			baselineElements:   []*storage.BaselineElement{fixtures.GetBaselineElement("Name One")},
			baselineExists:     true,
			baselineLocked:     false,
			expectedSuspicious: false,
		},
		{
			name: "Empty locked baseline",
			nameContainerGroup: v1.ProcessNameAndContainerNameGroup_builder{
				Name:          "Name One",
				ContainerName: "Container One",
			}.Build(),
			baselineElements:   []*storage.BaselineElement{},
			baselineExists:     true,
			baselineLocked:     true,
			expectedSuspicious: true,
		},
		{
			name: "Empty unlocked baseline",
			nameContainerGroup: v1.ProcessNameAndContainerNameGroup_builder{
				Name:          "Name One",
				ContainerName: "Container One",
			}.Build(),
			baselineElements:   []*storage.BaselineElement{},
			baselineExists:     true,
			baselineLocked:     false,
			expectedSuspicious: false,
		},
	}
	hasReadCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension, resources.Deployment)))
	mockCtrl := gomock.NewController(t)
	deployments := deploymentMocks.NewMockDataStore(mockCtrl)
	baselines := mocks.NewMockDataStore(mockCtrl)
	service := serviceImpl{deployments: deployments, baselines: baselines}
	testClusterID := "Test"
	testNamespace := "Test"
	testDeploymentID := "Test"
	testStart := protocompat.TimestampNow()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var baseline *storage.ProcessBaseline
			if c.baselineExists {
				baseline = fixtures.GetProcessBaseline()
				baseline.SetElements(c.baselineElements)
				if c.baselineLocked {
					baseline.SetUserLockedTimestamp(testStart)
				}
			}
			deployment := &storage.Deployment{}
			deployment.SetClusterId(testClusterID)
			deployment.SetNamespace(testNamespace)
			deployment.SetId(testDeploymentID)
			key := &storage.ProcessBaselineKey{}
			key.SetClusterId(testClusterID)
			key.SetNamespace(testNamespace)
			key.SetDeploymentId(testDeploymentID)
			key.SetContainerName(c.nameContainerGroup.GetContainerName())
			deployments.EXPECT().GetDeployment(gomock.Any(), testDeploymentID).Return(deployment, true, nil)
			baselines.EXPECT().GetProcessBaseline(gomock.Any(), key).Return(baseline, true, nil)
			err := service.setSuspicious(hasReadCtx, []*v1.ProcessNameAndContainerNameGroup{c.nameContainerGroup}, testDeploymentID)
			assert.NoError(t, err)
			assert.Equal(t, c.expectedSuspicious, c.nameContainerGroup.GetSuspicious())
		})
	}
}
