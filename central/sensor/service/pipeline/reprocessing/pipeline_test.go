package reprocessing

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	lifecycleMocks "github.com/stackrox/rox/central/detection/lifecycle/mocks"
	reprocessorMocks "github.com/stackrox/rox/central/reprocessor/mocks"
	riskManagerMocks "github.com/stackrox/rox/central/risk/manager/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
)

func TestRunCallsReprocessDeploymentRisk(t *testing.T) {
	ctrl := gomock.NewController(t)
	deployments := deploymentMocks.NewMockDataStore(ctrl)
	manager := lifecycleMocks.NewMockManager(ctrl)
	riskMgr := riskManagerMocks.NewMockManager(ctrl)
	reprocessor := reprocessorMocks.NewMockLoop(ctrl)

	p := &pipelineImpl{
		deployments:     deployments,
		riskManager:     riskMgr,
		riskReprocessor: reprocessor,
		manager:         manager,
		riskSemaphore:   semaphore.NewWeighted(15),
	}

	depID := uuid.NewV4().String()
	dep := &storage.Deployment{Id: depID, Name: "test"}
	deployments.EXPECT().GetDeployment(gomock.Any(), depID).Return(dep, true, nil)
	riskMgr.EXPECT().ReprocessDeploymentRisk(dep)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_ReprocessDeployment{
					ReprocessDeployment: &central.ReprocessDeploymentRisk{
						DeploymentId: depID,
					},
				},
			},
		},
	}

	err := p.Run(context.Background(), "cluster-1", msg, nil)
	assert.NoError(t, err)
}

func TestRunThrottlesConcurrency(t *testing.T) {
	ctrl := gomock.NewController(t)
	deployments := deploymentMocks.NewMockDataStore(ctrl)
	manager := lifecycleMocks.NewMockManager(ctrl)
	riskMgr := riskManagerMocks.NewMockManager(ctrl)
	reprocessor := reprocessorMocks.NewMockLoop(ctrl)

	const maxConcurrency = 3
	p := &pipelineImpl{
		deployments:     deployments,
		riskManager:     riskMgr,
		riskReprocessor: reprocessor,
		manager:         manager,
		riskSemaphore:   semaphore.NewWeighted(maxConcurrency),
	}

	// Track peak concurrency inside ReprocessDeploymentRisk.
	var currentConcurrency atomic.Int32
	var peakConcurrency atomic.Int32
	var completed atomic.Int32

	const numDeployments = 20

	// Set up mocks: each ReprocessDeploymentRisk call sleeps briefly to simulate DB work.
	for i := 0; i < numDeployments; i++ {
		depID := uuid.NewV4().String()
		dep := &storage.Deployment{Id: depID, Name: "test"}
		deployments.EXPECT().GetDeployment(gomock.Any(), depID).Return(dep, true, nil)
		riskMgr.EXPECT().ReprocessDeploymentRisk(dep).Do(func(_ *storage.Deployment) {
			cur := currentConcurrency.Add(1)
			// Update peak
			for {
				peak := peakConcurrency.Load()
				if cur <= peak {
					break
				}
				if peakConcurrency.CompareAndSwap(peak, cur) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			currentConcurrency.Add(-1)
			completed.Add(1)
		})

		msg := &central.MsgFromSensor{
			Msg: &central.MsgFromSensor_Event{
				Event: &central.SensorEvent{
					Resource: &central.SensorEvent_ReprocessDeployment{
						ReprocessDeployment: &central.ReprocessDeploymentRisk{
							DeploymentId: depID,
						},
					},
				},
			},
		}

		go func() {
			err := p.Run(context.Background(), "cluster-1", msg, nil)
			assert.NoError(t, err)
		}()
	}

	// Wait for all goroutines to finish.
	require.Eventually(t, func() bool {
		return completed.Load() == numDeployments
	}, 10*time.Second, 10*time.Millisecond)

	// The peak concurrency should not exceed our limit.
	assert.LessOrEqual(t, peakConcurrency.Load(), int32(maxConcurrency),
		"peak concurrency (%d) should not exceed the semaphore limit (%d)",
		peakConcurrency.Load(), maxConcurrency)
	t.Logf("Peak concurrency: %d (limit: %d)", peakConcurrency.Load(), maxConcurrency)
}

func TestRunRespectsContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	deployments := deploymentMocks.NewMockDataStore(ctrl)
	manager := lifecycleMocks.NewMockManager(ctrl)
	riskMgr := riskManagerMocks.NewMockManager(ctrl)
	reprocessor := reprocessorMocks.NewMockLoop(ctrl)

	// Semaphore with capacity 1, pre-acquired so the next Acquire blocks.
	sem := semaphore.NewWeighted(1)
	require.NoError(t, sem.Acquire(context.Background(), 1))

	p := &pipelineImpl{
		deployments:     deployments,
		riskManager:     riskMgr,
		riskReprocessor: reprocessor,
		manager:         manager,
		riskSemaphore:   sem,
	}

	// Note: no GetDeployment expectation because the semaphore is acquired first.
	// With a cancelled context and a full semaphore, Acquire returns immediately
	// with an error before any DB call is made.

	depID := uuid.NewV4().String()
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_ReprocessDeployment{
					ReprocessDeployment: &central.ReprocessDeploymentRisk{
						DeploymentId: depID,
					},
				},
			},
		},
	}

	// Cancel context so the semaphore Acquire fails.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.Run(ctx, "cluster-1", msg, nil)
	assert.Error(t, err, "Run should return an error when context is cancelled")

	// Release the pre-acquired semaphore to clean up.
	sem.Release(1)
}
