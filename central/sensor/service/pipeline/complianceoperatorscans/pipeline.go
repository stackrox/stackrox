package complianceoperatorscans

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/manager"
	"github.com/stackrox/rox/central/complianceoperator/scans/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/set"
)

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)
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

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	existingIDs := set.NewStringSet()
	walkFn := func() error {
		existingIDs.Clear()
		return s.datastore.Walk(ctx, func(scan *storage.ComplianceOperatorScan) error {
			if scan.GetClusterId() == clusterID {
				existingIDs.Add(scan.GetId())
			}
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return err
	}
	store := storeMap.Get((*central.SensorEvent_ComplianceOperatorScan)(nil))
	return reconciliation.Perform(store, existingIDs, "complianceoperatorscans", func(id string) error {
		return s.datastore.Delete(ctx, id)
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetComplianceOperatorScan() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(_ context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ComplianceOperatorScan)

	event := msg.GetEvent()
	scan := event.GetComplianceOperatorScan()
	scan.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.manager.DeleteScan(scan)
	default:
		return s.manager.AddScan(scan)
	}
}

func (s *pipelineImpl) OnFinish(_ string) {}
