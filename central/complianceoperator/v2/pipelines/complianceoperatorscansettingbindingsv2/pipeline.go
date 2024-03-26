package complianceoperatorscansettingbindingsv2

import (
	"context"

	v2Datastore "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore"
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
	return NewPipeline(v2Datastore.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(v2ScanSettingBindingDatastore v2Datastore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		v2ScanSettingBindingDatastore: v2ScanSettingBindingDatastore,
	}
}

type pipelineImpl struct {
	v2ScanSettingBindingDatastore v2Datastore.DataStore
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}

	existingIDs := set.NewStringSet()
	scanSettingBindings, err := s.v2ScanSettingBindingDatastore.GetScanSettingBindingsByCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	for _, binding := range scanSettingBindings {
		// The UID is used for reconciliation
		existingIDs.Add(binding.GetId())
	}

	store := storeMap.Get((*central.SensorEvent_ComplianceOperatorScanSettingBindingV2)(nil))
	return reconciliation.Perform(store, existingIDs, "complianceoperatorscansettingbindingv2", func(id string) error {
		return s.v2ScanSettingBindingDatastore.DeleteScanSettingBinding(ctx, id)
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetComplianceOperatorScanSettingBindingV2() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ComplianceOperatorScanSettingBinding)

	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}

	event := msg.GetEvent()
	binding := event.GetComplianceOperatorScanSettingBindingV2()

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.v2ScanSettingBindingDatastore.DeleteScanSettingBinding(ctx, binding.GetId())
	default:
		return s.v2ScanSettingBindingDatastore.UpsertScanSettingBinding(ctx, internaltov2storage.ComplianceOperatorScanSettingBindingObject(binding, clusterID))
	}
}

func (s *pipelineImpl) OnFinish(_ string) {}
