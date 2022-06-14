package complianceoperatorprofiles

import (
	"context"

	"github.com/stackrox/stackrox/central/complianceoperator/manager"
	"github.com/stackrox/stackrox/central/complianceoperator/profiles/datastore"
	countMetrics "github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/central/sensor/service/common"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/pkg/set"
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(datastore.Singleton(), manager.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(datastore datastore.DataStore, manager manager.Manager) pipeline.Fragment {
	return &pipelineImpl{
		datastore: datastore,
		manager:   manager,
	}
}

type pipelineImpl struct {
	datastore datastore.DataStore
	manager   manager.Manager
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	existingIDs := set.NewStringSet()
	err := s.datastore.Walk(ctx, func(profile *storage.ComplianceOperatorProfile) error {
		if profile.GetClusterId() == clusterID {
			existingIDs.Add(profile.GetId())
		}
		return nil
	})
	if err != nil {
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

func (s *pipelineImpl) OnFinish(_ string) {}
