package service

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"
)

func TestGenerateTimeseries(t *testing.T) {
	alerts := []*v1.Alert{
		{
			Id: "id1",
			Time: &timestamp.Timestamp{
				Seconds: 1,
			},
			Stale: true,
			MarkedStale: &timestamp.Timestamp{
				Seconds: 8,
			},
		},
		{
			Id: "id2",
			Time: &timestamp.Timestamp{
				Seconds: 6,
			},
		},
	}
	expectedEvents := []*v1.Event{
		{
			Time:   1000,
			Id:     "id1",
			Action: v1.Action_CREATED,
		},
		{
			Time:   6000,
			Id:     "id2",
			Action: v1.Action_CREATED,
		},
		{
			Time:   8000,
			Id:     "id1",
			Action: v1.Action_REMOVED,
		},
	}
	assert.Empty(t, getEventsFromAlerts(nil))
	assert.Equal(t, expectedEvents, getEventsFromAlerts(alerts))
}
