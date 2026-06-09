package connection

import (
	"testing"

	hashManager "github.com/stackrox/rox/central/hash/manager"
	pipelineMock "github.com/stackrox/rox/central/sensor/service/pipeline/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDuplicateSyncedEventDoesNotPanic(t *testing.T) {
	const clusterID = "duplicate-synced-cluster"

	ctrl := gomock.NewController(t)
	mockPipeline := pipelineMock.NewMockClusterPipeline(ctrl)
	var reconcileCalls int
	mockPipeline.EXPECT().Reconcile(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(_ any, _ any) error {
		reconcileCalls++
		return nil
	})

	stopSig := concurrency.NewErrorSignal()
	deduper := hashManager.NewDeduper(nil, clusterID)
	deduper.StartSync()

	handler := newSensorEventHandler(
		&storage.Cluster{Id: clusterID, Name: "duplicate-synced-cluster"},
		"regression-sensor-version",
		mockPipeline,
		nil,
		&stopSig,
		deduper,
		NewInitSyncManager(),
	)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id: "duplicate-synced-regression",
				Resource: &central.SensorEvent_Synced{
					Synced: &central.SensorEvent_ResourcesSynced{
						UnchangedIds: []string{
							"Deployment:deployment-1",
							"Pod:pod-1",
						},
					},
				},
			},
		},
	}

	handler.addMultiplexed(t.Context(), msg)

	assert.NotPanics(t, func() {
		handler.addMultiplexed(t.Context(), msg.CloneVT())
	})
	assert.Equal(t, 1, reconcileCalls)
	assert.True(t, handler.reconciliationMap.IsClosed())
}
