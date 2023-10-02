package complianceoperatorresults

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/checkresults/datastore"
	v2 "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/set"
)

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	if features.ComplianceEnhancements.Enabled() {
		return NewPipeline(datastore.Singleton(), v2.Singleton())
	}

	return NewPipeline(datastore.Singleton(), nil)
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(datastore datastore.DataStore, v2Datastore v2.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		datastore:   datastore,
		v2Datastore: v2Datastore,
	}
}

type pipelineImpl struct {
	datastore   datastore.DataStore
	v2Datastore v2.DataStore
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	if features.ComplianceEnhancements.Enabled() {
		existingIDs := set.NewStringSet()
		// TODO:  Search to get results for the cluster
		// TODO:  Loop through the results and add the IDs.

		store := storeMap.Get((*central.SensorEvent_ComplianceOperatorResult)(nil))

		return reconciliation.Perform(store, existingIDs, "complianceoperatorcheckresults", func(id string) error {
			return s.v2Datastore.Delete(ctx, id)
		})
	}

	existingIDs := set.NewStringSet()
	walkFn := func() error {
		existingIDs.Clear()
		return s.datastore.Walk(ctx, func(check *storage.ComplianceOperatorCheckResult) error {
			if check.GetClusterId() == clusterID {
				existingIDs.Add(check.GetId())
			}
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return err
	}

	store := storeMap.Get((*central.SensorEvent_ComplianceOperatorResult)(nil))
	return reconciliation.Perform(store, existingIDs, "complianceoperatorcheckresults", func(id string) error {
		return s.datastore.Delete(ctx, id)
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetComplianceOperatorResult() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ComplianceOperatorCheckResult)

	event := msg.GetEvent()
	checkResult := event.GetComplianceOperatorResult()
	checkResult.ClusterId = clusterID

	if features.ComplianceEnhancements.Enabled() {
		switch event.GetAction() {
		case central.ResourceAction_REMOVE_RESOURCE:
			// use V2 datastore
			return s.v2Datastore.Delete(ctx, event.GetId())
		default:
			return s.v2Datastore.Upsert(ctx, convertSensorMsgToV2Storage(checkResult, clusterID))
		}
	}

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.datastore.Delete(ctx, event.GetId())
	default:
		return s.datastore.Upsert(ctx, checkResult)
	}
}

func (s *pipelineImpl) OnFinish(_ string) {}

func convertSensorMsgToV2Storage(sensorData *storage.ComplianceOperatorCheckResult, clusterID string) *storage.ComplianceOperatorCheckResultV2 {
	return &storage.ComplianceOperatorCheckResultV2{
		Id:           sensorData.GetId(),
		CheckId:      sensorData.GetCheckId(),
		CheckName:    sensorData.GetCheckName(),
		ClusterId:    clusterID,
		Status:       2, // TODO convert this
		Severity:     3, // TODO convert this
		Description:  sensorData.GetDescription(),
		Instructions: sensorData.GetInstructions(),
		Labels:       sensorData.GetLabels(),
		Annotations:  sensorData.GetAnnotations(),
		CreatedTime:  nil, // TODO pull this from labels/annotations
		Standard:     "",  // TODO pull this form labels/annotations
		Control:      "",  // TODO pull this form labels/annotations
		ScanId:       "",  // TODO pull this form labels/annotations probably look it up from scan name
	}
}
