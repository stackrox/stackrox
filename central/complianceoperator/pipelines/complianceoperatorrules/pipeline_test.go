package complianceoperatorrules

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	managerMocks "github.com/stackrox/rox/central/complianceoperator/manager/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
)

func makeRuleMsg(ruleID, ruleName string) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     ruleID,
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorRule{
					ComplianceOperatorRule: &storage.ComplianceOperatorRule{
						Id:   ruleID,
						Name: ruleName,
						Annotations: map[string]string{
							v1alpha1.RuleIDAnnotationKey: ruleName,
						},
					},
				},
			},
		},
	}
}

func TestRunCallsAddRule(t *testing.T) {
	ctrl := gomock.NewController(t)
	mgr := managerMocks.NewMockManager(ctrl)
	p := &pipelineImpl{
		manager:   mgr,
		semaphore: semaphore.NewWeighted(5),
	}

	mgr.EXPECT().AddRule(gomock.Any()).Return(nil)

	err := p.Run(context.Background(), "cluster-1", makeRuleMsg(uuid.NewV4().String(), "test-rule"), nil)
	assert.NoError(t, err)
}

func TestThrottlesConcurrency(t *testing.T) {
	ctrl := gomock.NewController(t)
	mgr := managerMocks.NewMockManager(ctrl)

	const maxConcurrency = 3
	p := &pipelineImpl{
		manager:   mgr,
		semaphore: semaphore.NewWeighted(maxConcurrency),
	}

	var currentConcurrency atomic.Int32
	var peakConcurrency atomic.Int32
	var completed atomic.Int32

	const numRules = 20

	for i := 0; i < numRules; i++ {
		mgr.EXPECT().AddRule(gomock.Any()).Do(func(_ *storage.ComplianceOperatorRule) {
			cur := currentConcurrency.Add(1)
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
		}).Return(nil)

		msg := makeRuleMsg(uuid.NewV4().String(), "test-rule")

		go func() {
			err := p.Run(context.Background(), "cluster-1", msg, nil)
			assert.NoError(t, err)
		}()
	}

	require.Eventually(t, func() bool {
		return completed.Load() == numRules
	}, 10*time.Second, 10*time.Millisecond)

	assert.LessOrEqual(t, peakConcurrency.Load(), int32(maxConcurrency),
		"peak concurrency (%d) should not exceed the semaphore limit (%d)",
		peakConcurrency.Load(), maxConcurrency)
	t.Logf("Peak concurrency: %d (limit: %d)", peakConcurrency.Load(), maxConcurrency)
}

func TestRespectsContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	mgr := managerMocks.NewMockManager(ctrl)

	sem := semaphore.NewWeighted(1)
	require.NoError(t, sem.Acquire(context.Background(), 1))

	p := &pipelineImpl{
		manager:   mgr,
		semaphore: sem,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.Run(ctx, "cluster-1", makeRuleMsg(uuid.NewV4().String(), "test-rule"), nil)
	assert.Error(t, err, "Run should return an error when context is cancelled")

	sem.Release(1)
}
