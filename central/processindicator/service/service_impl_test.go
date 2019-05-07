package service

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/processwhitelist/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stretchr/testify/assert"
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

func TestWhitelistCheck(t *testing.T) {
	var cases = []struct {
		name               string
		nameContainerGroup *v1.ProcessNameAndContainerNameGroup
		whitelistElements  []*storage.WhitelistElement
		whitelistExists    bool
		whitelistLocked    bool
		expectedSuspicious bool
	}{
		{
			name: "On the whitelist",
			nameContainerGroup: &v1.ProcessNameAndContainerNameGroup{
				Name:          "Name One",
				ContainerName: "Container One",
			},
			whitelistElements:  []*storage.WhitelistElement{fixtures.GetWhitelistElement("Name One")},
			whitelistExists:    true,
			whitelistLocked:    true,
			expectedSuspicious: false,
		},
		{
			name: "Not on the whitelist",
			nameContainerGroup: &v1.ProcessNameAndContainerNameGroup{
				Name:          "Name Two",
				ContainerName: "Container One",
			},
			whitelistElements:  []*storage.WhitelistElement{fixtures.GetWhitelistElement("Name One")},
			whitelistExists:    true,
			whitelistLocked:    true,
			expectedSuspicious: true,
		},
		{
			name: "No whitelist",
			nameContainerGroup: &v1.ProcessNameAndContainerNameGroup{
				Name:          "Name One",
				ContainerName: "Container One",
			},
			whitelistExists:    false,
			expectedSuspicious: false,
		},
		{
			name: "Unlocked whitelist",
			nameContainerGroup: &v1.ProcessNameAndContainerNameGroup{
				Name:          "Name Two",
				ContainerName: "Container One",
			},
			whitelistElements:  []*storage.WhitelistElement{fixtures.GetWhitelistElement("Name One")},
			whitelistExists:    true,
			whitelistLocked:    false,
			expectedSuspicious: false,
		},
		{
			name: "Empty locked whitelist",
			nameContainerGroup: &v1.ProcessNameAndContainerNameGroup{
				Name:          "Name One",
				ContainerName: "Container One",
			},
			whitelistElements:  []*storage.WhitelistElement{},
			whitelistExists:    true,
			whitelistLocked:    true,
			expectedSuspicious: true,
		},
		{
			name: "Empty unlocked whitelist",
			nameContainerGroup: &v1.ProcessNameAndContainerNameGroup{
				Name:          "Name One",
				ContainerName: "Container One",
			},
			whitelistElements:  []*storage.WhitelistElement{},
			whitelistExists:    true,
			whitelistLocked:    false,
			expectedSuspicious: false,
		},
	}
	mockCtrl := gomock.NewController(t)
	whitelists := mocks.NewMockDataStore(mockCtrl)
	service := serviceImpl{whitelists: whitelists}
	testDeploymentID := "Test"
	testStart := types.TimestampNow()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var whitelist *storage.ProcessWhitelist
			if c.whitelistExists {
				whitelist = fixtures.GetProcessWhitelist()
				whitelist.Elements = c.whitelistElements
				if c.whitelistLocked {
					whitelist.UserLockedTimestamp = testStart
				}
			}
			key := &storage.ProcessWhitelistKey{DeploymentId: testDeploymentID, ContainerName: c.nameContainerGroup.ContainerName}
			whitelists.EXPECT().GetProcessWhitelist(gomock.Any(), key).Return(whitelist, nil)
			err := service.setSuspicious(context.TODO(), []*v1.ProcessNameAndContainerNameGroup{c.nameContainerGroup}, testDeploymentID)
			assert.NoError(t, err)
			assert.Equal(t, c.expectedSuspicious, c.nameContainerGroup.Suspicious)
		})
	}
}
