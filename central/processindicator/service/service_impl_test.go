package service

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

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
