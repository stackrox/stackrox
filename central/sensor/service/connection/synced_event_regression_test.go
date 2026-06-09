package connection

import (
	"testing"

	hashManager "github.com/stackrox/rox/central/hash/manager"
	pipelineMock "github.com/stackrox/rox/central/sensor/service/pipeline/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// Malformed unchanged_ids should be ignored rather than crashing the Synced
// event reconciliation path. Valid entries still need to reconcile, and sync
// bookkeeping still needs to complete.
func TestSyncedEventMalformedUnchangedIDsDoesNotPanic(t *testing.T) {
	const clusterID = "regression-cluster"

	ctrl := gomock.NewController(t)
	mockPipeline := pipelineMock.NewMockClusterPipeline(ctrl)
	var reconcileCalls int
	mockPipeline.EXPECT().Reconcile(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(_ any, _ any) error {
		reconcileCalls++
		return nil
	})

	stopSig := concurrency.NewErrorSignal()
	initSyncMgr := &initSyncManager{
		maxSensors: 1,
		sensors:    set.NewStringSet(clusterID),
	}
	deduper := hashManager.NewDeduper(nil, clusterID)
	deduper.StartSync()

	handler := newSensorEventHandler(
		&storage.Cluster{Id: clusterID, Name: "regression-cluster"},
		"regression-sensor-version",
		mockPipeline,
		nil,
		&stopSig,
		deduper,
		initSyncMgr,
	)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id: "synced-regression",
				Resource: &central.SensorEvent_Synced{
					Synced: &central.SensorEvent_ResourcesSynced{
						UnchangedIds: []string{
							"Deployment:deployment-2",
							"definitely-not-a-deduper-key",
							"AlertResults:alert-1",
						},
					},
				},
			},
		},
	}

	assert.NotPanics(t, func() {
		handler.addMultiplexed(t.Context(), msg)
	})
	assert.Equal(t, 1, reconcileCalls)
	assert.True(t, handler.reconciliationMap.IsClosed())
	assert.False(t, initSyncMgr.sensors.Contains(clusterID))
	assert.Len(t, handler.workerQueues, 0)
}
