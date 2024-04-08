package complianceoperatorscansv2

import (
	"context"

	"github.com/pkg/errors"
	v2 "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/central/convert/internaltov2storage"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/set"
)

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(v2.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(v2Datastore v2.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		v2Datastore: v2Datastore,
	}
}

type pipelineImpl struct {
	v2Datastore v2.DataStore
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}

	existingIDs := set.NewStringSet()
	scanObjects, err := s.v2Datastore.GetScansByCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	for _, scanObject := range scanObjects {
		// The UID is used for reconciliation
		existingIDs.Add(scanObject.GetId())
	}

	store := storeMap.Get((*central.SensorEvent_ComplianceOperatorScanV2)(nil))
	return reconciliation.Perform(store, existingIDs, "complianceoperatorscansv2", func(id string) error {
		return s.v2Datastore.DeleteScan(ctx, id)
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetComplianceOperatorScanV2() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ComplianceOperatorScanV2)

	if !features.ComplianceEnhancements.Enabled() {
		return errors.New("Next gen compliance is disabled.  Message unexpected.")
	}

	event := msg.GetEvent()
	complianceScanObject := event.GetComplianceOperatorScanV2()

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.v2Datastore.DeleteScan(ctx, event.GetId())
	default:
		return s.v2Datastore.UpsertScan(ctx, internaltov2storage.ComplianceOperatorScanObject(complianceScanObject, clusterID))
	}
}

func (s *pipelineImpl) OnFinish(_ string) {}
