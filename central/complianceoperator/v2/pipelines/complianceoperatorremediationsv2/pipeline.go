package complianceoperatorremediationsv2

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/remediations/datastore"
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
	return NewPipeline(datastore.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(datastore datastore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		datastore: datastore,
	}
}

type pipelineImpl struct {
	datastore datastore.DataStore
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	if !features.ComplianceRemediationV2.Enabled() || !features.ComplianceEnhancements.Enabled() {
		return nil
	}
	existingIDs := set.NewStringSet()
	remediations, err := s.datastore.GetRemediationsByCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	for _, remediation := range remediations {
		// The UID is used for reconciliation
		existingIDs.Add(remediation.GetId())
	}

	store := storeMap.Get((*central.SensorEvent_ComplianceOperatorRemediationV2)(nil))
	return reconciliation.Perform(store, existingIDs, "complianceoperatorremediationv2", func(id string) error {
		return s.datastore.DeleteRemediation(ctx, id)
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetComplianceOperatorRemediationV2() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	if !features.ComplianceRemediationV2.Enabled() || !features.ComplianceEnhancements.Enabled() {
		return nil
	}

	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ComplianceOperatorRemediationV2)

	event := msg.GetEvent()
	remediation := event.GetComplianceOperatorRemediationV2()

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.datastore.DeleteRemediation(ctx, remediation.GetId())
	default:
		return s.datastore.UpsertRemediation(ctx, internaltov2storage.ComplianceOperatorRemediation(remediation, clusterID))
	}
}

func (s *pipelineImpl) OnFinish(_ string) {}
