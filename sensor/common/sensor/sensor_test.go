package sensor

import (
	"testing"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/assert"
)

type recordingNotifiable struct {
	notifications []common.SensorComponentEvent
}

func (r *recordingNotifiable) Notify(event common.SensorComponentEvent) {
	r.notifications = append(r.notifications, event)
}

func TestTriggerOfflineMode(t *testing.T) {
	tests := map[string]struct {
		initialState      common.SensorComponentEvent
		wantState         common.SensorComponentEvent
		wantNotifications []common.SensorComponentEvent
	}{
		"should emit synthetic reconnect sequence when sensor is online": {
			initialState: common.SensorComponentEventCentralReachable,
			wantState:    common.SensorComponentEventSyncFinished,
			wantNotifications: []common.SensorComponentEvent{
				common.SensorComponentEventOfflineMode,
				common.SensorComponentEventCentralReachableHTTP,
				common.SensorComponentEventCentralReachable,
				common.SensorComponentEventSyncFinished,
			},
		},
		"should not emit notifications when sensor is already offline": {
			initialState:      common.SensorComponentEventOfflineMode,
			wantState:         common.SensorComponentEventOfflineMode,
			wantNotifications: nil,
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			notifiable := &recordingNotifiable{}
			sensor := &Sensor{
				currentState:    testCase.initialState,
				currentStateMtx: &sync.Mutex{},
				notifyList:      []common.Notifiable{notifiable},
			}

			sensor.TriggerOfflineMode("unit test")

			assert.Equal(t, testCase.wantNotifications, notifiable.notifications)
			assert.Equal(t, testCase.wantState, sensor.currentState)
		})
	}
}
