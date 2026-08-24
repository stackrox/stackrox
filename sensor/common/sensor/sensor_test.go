package sensor

import (
	"testing"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/assert"
)

type fakeNotifiable struct {
	events []common.SensorComponentEvent
}

func (f *fakeNotifiable) Notify(e common.SensorComponentEvent) {
	f.events = append(f.events, e)
}

func TestTriggerOfflineMode(t *testing.T) {
	tests := map[string]struct {
		initialState   common.SensorComponentEvent
		expectedEvents []common.SensorComponentEvent
		expectedState  common.SensorComponentEvent
	}{
		"already offline is a no-op": {
			initialState:   common.SensorComponentEventOfflineMode,
			expectedEvents: nil,
			expectedState:  common.SensorComponentEventOfflineMode,
		},
		"emits full reconnect sequence but persists state only up to CentralReachableHTTP": {
			initialState: common.SensorComponentEventCentralReachable,
			expectedEvents: []common.SensorComponentEvent{
				common.SensorComponentEventOfflineMode,
				common.SensorComponentEventCentralReachableHTTP,
				common.SensorComponentEventCentralReachable,
				common.SensorComponentEventSyncFinished,
			},
			expectedState: common.SensorComponentEventCentralReachableHTTP,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			notified := &fakeNotifiable{}
			s := &Sensor{
				currentStateMtx: &sync.Mutex{},
				currentState:    tc.initialState,
				notifyList:      []common.Notifiable{notified},
			}

			s.TriggerOfflineMode("test")

			assert.Equal(t, tc.expectedEvents, notified.events)
			assert.Equal(t, tc.expectedState, s.currentState)
		})
	}
}
