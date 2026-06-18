package complianceoperatorprofiles

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/manager"
	"github.com/stackrox/rox/central/complianceoperator/profiles/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/set"
	"golang.org/x/sync/semaphore"
)

var (
	log = logging.LoggerForModule()

	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(datastore.Singleton(), manager.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(datastore datastore.DataStore, manager manager.Manager) pipeline.Fragment {
	maxConcurrency := int64(env.ComplianceV1MaxConcurrency.IntegerSetting())
	return &pipelineImpl{
		datastore: datastore,
		manager:   manager,
		semaphore: semaphore.NewWeighted(maxConcurrency),
	}
}

type pipelineImpl struct {
	datastore datastore.DataStore
	manager   manager.Manager
	semaphore *semaphore.Weighted
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	existingIDs := set.NewStringSet()
	walkFn := func() error {
		existingIDs.Clear()
		return s.datastore.Walk(ctx, func(profile *storage.ComplianceOperatorProfile) error {
			if profile.GetClusterId() == clusterID {
				existingIDs.Add(profile.GetId())
			}
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(ctx, walkFn); err != nil {
		return err
	}

	store := storeMap.Get((*central.SensorEvent_ComplianceOperatorProfile)(nil))
	return reconciliation.Perform(store, existingIDs, "complianceoperatorprofiles", func(id string) error {
		return s.datastore.Delete(ctx, id)
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetComplianceOperatorProfile() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ComplianceOperatorProfile)

	if err := s.acquireSemaphore(ctx); err != nil {
		return err
	}
	defer s.semaphore.Release(1)

	event := msg.GetEvent()
	profile := event.GetComplianceOperatorProfile()
	profile.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.manager.DeleteProfile(profile)
	default:
		return s.manager.AddProfile(profile)
	}
}

func (s *pipelineImpl) acquireSemaphore(ctx context.Context) error {
	waitTime := env.ComplianceV1SemaphoreWaitTime.DurationSetting()

	acquireCtx := ctx
	if waitTime > 0 {
		var cancel context.CancelFunc
		acquireCtx, cancel = context.WithTimeout(ctx, waitTime)
		defer cancel()
	}

	if err := s.semaphore.Acquire(acquireCtx, 1); err != nil {
		if ctx.Err() != nil {
			log.Debugf("Unable to acquire semaphore for compliance profile update: %v", err)
		} else if errors.Is(err, context.DeadlineExceeded) {
			log.Warnf("Timed out waiting to process compliance profile (waited %v): %v", waitTime, err)
		} else {
			log.Errorf("Unexpected error acquiring compliance profile semaphore: %v", err)
		}
		return err
	}
	return nil
}

func (s *pipelineImpl) OnFinish(_ string) {}
