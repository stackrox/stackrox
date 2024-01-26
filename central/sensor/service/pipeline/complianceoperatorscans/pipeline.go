package complianceoperatorscans

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/manager"
	"github.com/stackrox/rox/central/complianceoperator/scans/datastore"
	v2 "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/central/convert/internaltov1storage"
	"github.com/stackrox/rox/central/convert/internaltov2storage"
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
		return NewPipeline(datastore.Singleton(), manager.Singleton(), v2.Singleton())
	}
	return NewPipeline(datastore.Singleton(), manager.Singleton(), nil)
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(datastore datastore.DataStore, manager manager.Manager, v2Datastore v2.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		datastore:   datastore,
		manager:     manager,
		v2Datastore: v2Datastore,
	}
}

type pipelineImpl struct {
	datastore   datastore.DataStore
	manager     manager.Manager
	v2Datastore v2.DataStore
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

	// For now if nextgen compliance is enabled, we have to reconcile both versions of compliance.
	if features.ComplianceEnhancements.Enabled() {
		scanObjects, err := s.v2Datastore.GetScansByCluster(ctx, clusterID)
		if err != nil {
			return err
		}

		for _, scanObject := range scanObjects {
			// The UID is used for reconciliation
			existingIDs.Add(scanObject.GetId())
		}
	}

	store := storeMap.Get((*central.SensorEvent_ComplianceOperatorScan)(nil))
	return reconciliation.Perform(store, existingIDs, "complianceoperatorscans", func(id string) error {
		if features.ComplianceEnhancements.Enabled() {
			if err := s.v2Datastore.DeleteScan(ctx, id); err != nil {
				return err
			}
		}

		return s.datastore.Delete(ctx, id)
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	if features.ComplianceEnhancements.Enabled() {
		return msg.GetEvent().GetComplianceOperatorScanV2() != nil || msg.GetEvent().GetComplianceOperatorScan() != nil
	}

	return msg.GetEvent().GetComplianceOperatorScan() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ComplianceOperatorScan)

	event := msg.GetEvent()

	// If a sensor sends in a v1 compliance message we will still process it the v1 way in the event
	// a sensor is not updated or does not have the flag on.
	switch event.Resource.(type) {
	case *central.SensorEvent_ComplianceOperatorScan:
		return s.processComplianceScan(ctx, event, clusterID)
	case *central.SensorEvent_ComplianceOperatorScanV2:
		if !features.ComplianceEnhancements.Enabled() {
			return errors.New("UNEXPECTED: Next gen compliance is disabled")
		}
		return s.processV2ComplianceScan(ctx, event, clusterID)
	}

	return errors.Errorf("unexpected message %T.", event.Resource)
}

func (s *pipelineImpl) OnFinish(_ string) {}

func (s *pipelineImpl) processComplianceScan(ctx context.Context, event *central.SensorEvent, clusterID string) error {
	complianceScanObject := event.GetComplianceOperatorScan()
	complianceScanObject.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.datastore.Delete(ctx, event.GetId())
	default:
		return s.datastore.Upsert(ctx, complianceScanObject)
	}
}

func (s *pipelineImpl) processV2ComplianceScan(ctx context.Context, event *central.SensorEvent, clusterID string) error {
	complianceScanObject := event.GetComplianceOperatorScanV2()

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		// V1 still needs to function so remove it too
		if err := s.datastore.Delete(ctx, event.GetId()); err != nil {
			return err
		}

		// use V2 datastore
		return s.v2Datastore.DeleteScan(ctx, event.GetId())
	default:
		// Still need to store the V1 version to maintain both
		if err := s.datastore.Upsert(ctx, internaltov1storage.ComplianceOperatorScanObject(complianceScanObject, clusterID)); err != nil {
			return err
		}

		return s.v2Datastore.UpsertScan(ctx, internaltov2storage.ComplianceOperatorScanObject(complianceScanObject, clusterID))
	}
}
