package service

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestIndicatorsToGroupedResponses(t *testing.T) {
	var cases = []struct {
		name       string
		indicators []*v1.ProcessIndicator
		nameGroups []*v1.ProcessNameGroup
	}{
		{
			name: "test grouping",
			indicators: []*v1.ProcessIndicator{
				{
					Signal: &v1.ProcessSignal{
						Id:           "1",
						ExecFilePath: "cat",
						Args:         "hello",
					},
				},
				{
					Signal: &v1.ProcessSignal{
						Id:           "2",
						ExecFilePath: "cat",
						Args:         "hello",
					},
				},
				{
					Signal: &v1.ProcessSignal{
						Id:           "3",
						ExecFilePath: "cat",
						Args:         "boo",
					},
				},
				{
					Signal: &v1.ProcessSignal{
						Id:           "4",
						ExecFilePath: "blah",
						Args:         "boo",
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
							Signals: []*v1.ProcessIndicator{
								{
									Signal: &v1.ProcessSignal{
										Id:           "4",
										ExecFilePath: "blah",
										Args:         "boo",
									},
								},
							},
						},
					},
				},
				{
					Name:          "cat",
					TimesExecuted: 3,
					Groups: []*v1.ProcessGroup{
						{
							Args: "boo",
							Signals: []*v1.ProcessIndicator{
								{
									Signal: &v1.ProcessSignal{
										Id:           "3",
										ExecFilePath: "cat",
										Args:         "boo",
									},
								},
							},
						},
						{
							Args: "hello",
							Signals: []*v1.ProcessIndicator{
								{
									Signal: &v1.ProcessSignal{
										Id:           "1",
										ExecFilePath: "cat",
										Args:         "hello",
									},
								},
								{
									Signal: &v1.ProcessSignal{
										Id:           "2",
										ExecFilePath: "cat",
										Args:         "hello",
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
			assert.Equal(t, c.nameGroups, indicatorsToGroupedResponses(c.indicators))
		})
	}
}
