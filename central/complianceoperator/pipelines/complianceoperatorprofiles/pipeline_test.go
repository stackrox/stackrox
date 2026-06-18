package complianceoperatorprofiles

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	managerMocks "github.com/stackrox/rox/central/complianceoperator/manager/mocks"
	v1ProfileMocks "github.com/stackrox/rox/central/complianceoperator/profiles/datastore/mocks"
	"github.com/stackrox/rox/central/convert/testutils"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite
	pipeline    *pipelineImpl
	manager     *managerMocks.MockManager
	v1ProfileDS *v1ProfileMocks.MockDataStore
	mockCtrl    *gomock.Controller
}

func (s *PipelineTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip tests when ComplianceEnhancements disabled")
		s.T().SkipNow()
	}
}

func (s *PipelineTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.manager = managerMocks.NewMockManager(s.mockCtrl)
	s.v1ProfileDS = v1ProfileMocks.NewMockDataStore(s.mockCtrl)
	s.pipeline = NewPipeline(s.v1ProfileDS, s.manager).(*pipelineImpl)
}

func (s *PipelineTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *PipelineTestSuite) TestRunV1Create() {
	ctx := context.Background()

	pipeline := NewPipeline(s.v1ProfileDS, s.manager)
	s.manager.EXPECT().AddProfile(testutils.GetProfileV1SensorMsg(s.T())).Return(nil).Times(1)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.ProfileUID,
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorProfile{
					ComplianceOperatorProfile: testutils.GetProfileV1SensorMsg(s.T()),
				},
			},
		},
	}

	err := pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunV1Delete() {
	ctx := context.Background()

	pipeline := NewPipeline(s.v1ProfileDS, s.manager)
	s.manager.EXPECT().DeleteProfile(testutils.GetProfileV1SensorMsg(s.T())).Return(nil).Times(1)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     testutils.ProfileUID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorProfile{
					ComplianceOperatorProfile: testutils.GetProfileV1SensorMsg(s.T()),
				},
			},
		},
	}

	err := pipeline.Run(ctx, fixtureconsts.Cluster1, msg, nil)
	s.NoError(err)
}

func (s *PipelineTestSuite) TestRunReconcileNoOp() {
	ctx := context.Background()

	pipeline := NewPipeline(s.v1ProfileDS, s.manager)

	s.v1ProfileDS.EXPECT().Walk(ctx, gomock.Any()).Return(nil).Times(1)

	err := pipeline.Reconcile(ctx, fixtureconsts.Cluster1, reconciliation.NewStoreMap())
	s.NoError(err)
}

func TestThrottlesConcurrency(t *testing.T) {
	ctrl := gomock.NewController(t)
	mgr := managerMocks.NewMockManager(ctrl)
	ds := v1ProfileMocks.NewMockDataStore(ctrl)

	const maxConcurrency = 3
	p := &pipelineImpl{
		datastore: ds,
		manager:   mgr,
		semaphore: semaphore.NewWeighted(maxConcurrency),
	}

	var currentConcurrency atomic.Int32
	var peakConcurrency atomic.Int32
	var completed atomic.Int32

	const numProfiles = 20

	for i := 0; i < numProfiles; i++ {
		profile := &storage.ComplianceOperatorProfile{
			Id:   uuid.NewV4().String(),
			Name: "profile",
		}
		mgr.EXPECT().AddProfile(gomock.Any()).Do(func(_ *storage.ComplianceOperatorProfile) {
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

		msg := &central.MsgFromSensor{
			Msg: &central.MsgFromSensor_Event{
				Event: &central.SensorEvent{
					Id:     profile.GetId(),
					Action: central.ResourceAction_CREATE_RESOURCE,
					Resource: &central.SensorEvent_ComplianceOperatorProfile{
						ComplianceOperatorProfile: profile,
					},
				},
			},
		}

		go func() {
			err := p.Run(context.Background(), "cluster-1", msg, nil)
			assert.NoError(t, err)
		}()
	}

	require.Eventually(t, func() bool {
		return completed.Load() == numProfiles
	}, 10*time.Second, 10*time.Millisecond)

	assert.LessOrEqual(t, peakConcurrency.Load(), int32(maxConcurrency),
		"peak concurrency (%d) should not exceed the semaphore limit (%d)",
		peakConcurrency.Load(), maxConcurrency)
	t.Logf("Peak concurrency: %d (limit: %d)", peakConcurrency.Load(), maxConcurrency)
}

func TestRespectsContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	mgr := managerMocks.NewMockManager(ctrl)
	ds := v1ProfileMocks.NewMockDataStore(ctrl)

	sem := semaphore.NewWeighted(1)
	require.NoError(t, sem.Acquire(context.Background(), 1))

	p := &pipelineImpl{
		datastore: ds,
		manager:   mgr,
		semaphore: sem,
	}

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     uuid.NewV4().String(),
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_ComplianceOperatorProfile{
					ComplianceOperatorProfile: &storage.ComplianceOperatorProfile{
						Id:   uuid.NewV4().String(),
						Name: "profile",
					},
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.Run(ctx, "cluster-1", msg, nil)
	assert.Error(t, err, "Run should return an error when context is cancelled")

	sem.Release(1)
}
