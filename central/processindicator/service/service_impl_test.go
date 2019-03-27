package service

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestIndicatorsToGroupedResponses(t *testing.T) {
	var cases = []struct {
		name       string
		indicators []*storage.ProcessIndicator
		nameGroups []*v1.ProcessNameGroup
	}{
		{
			name: "test grouping",
			indicators: []*storage.ProcessIndicator{
				{
					Signal: &storage.ProcessSignal{
						Id:           "1",
						ExecFilePath: "cat",
						Args:         "hello",
						ContainerId:  "A",
					},
				},
				{
					Signal: &storage.ProcessSignal{
						Id:           "2",
						ExecFilePath: "cat",
						Args:         "hello",
						ContainerId:  "B",
					},
				},
				{
					Signal: &storage.ProcessSignal{
						Id:           "3",
						ExecFilePath: "cat",
						Args:         "boo",
						ContainerId:  "A",
					},
				},
				{
					Signal: &storage.ProcessSignal{
						Id:           "4",
						ExecFilePath: "blah",
						Args:         "boo",
						ContainerId:  "C",
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
									Signal: &storage.ProcessSignal{
										Id:           "1",
										ExecFilePath: "cat",
										Args:         "hello",
										ContainerId:  "A",
									},
								},
								{
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
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.nameGroups, IndicatorsToGroupedResponses(c.indicators))
		})
	}
}
