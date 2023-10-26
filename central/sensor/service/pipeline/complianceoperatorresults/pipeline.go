package complianceoperatorresults

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/checkresults/datastore"
	v2 "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	scanConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)

	statusToV2 = map[central.ComplianceOperatorCheckResultV2_CheckStatus]storage.ComplianceOperatorCheckResultV2_CheckStatus{
		central.ComplianceOperatorCheckResultV2_UNSET:          storage.ComplianceOperatorCheckResultV2_UNSET,
		central.ComplianceOperatorCheckResultV2_PASS:           storage.ComplianceOperatorCheckResultV2_PASS,
		central.ComplianceOperatorCheckResultV2_FAIL:           storage.ComplianceOperatorCheckResultV2_FAIL,
		central.ComplianceOperatorCheckResultV2_ERROR:          storage.ComplianceOperatorCheckResultV2_ERROR,
		central.ComplianceOperatorCheckResultV2_INFO:           storage.ComplianceOperatorCheckResultV2_INFO,
		central.ComplianceOperatorCheckResultV2_MANUAL:         storage.ComplianceOperatorCheckResultV2_MANUAL,
		central.ComplianceOperatorCheckResultV2_NOT_APPLICABLE: storage.ComplianceOperatorCheckResultV2_NOT_APPLICABLE,
		central.ComplianceOperatorCheckResultV2_INCONSISTENT:   storage.ComplianceOperatorCheckResultV2_INCONSISTENT,
	}

	severityToV2 = map[central.ComplianceOperatorRuleSeverity]storage.RuleSeverity{
		central.ComplianceOperatorRuleSeverity_UNSET_RULE_SEVERITY:   storage.RuleSeverity_UNSET_RULE_SEVERITY,
		central.ComplianceOperatorRuleSeverity_UNKNOWN_RULE_SEVERITY: storage.RuleSeverity_UNKNOWN_RULE_SEVERITY,
		central.ComplianceOperatorRuleSeverity_INFO_RULE_SEVERITY:    storage.RuleSeverity_INFO_RULE_SEVERITY,
		central.ComplianceOperatorRuleSeverity_LOW_RULE_SEVERITY:     storage.RuleSeverity_LOW_RULE_SEVERITY,
		central.ComplianceOperatorRuleSeverity_MEDIUM_RULE_SEVERITY:  storage.RuleSeverity_MEDIUM_RULE_SEVERITY,
		central.ComplianceOperatorRuleSeverity_HIGH_RULE_SEVERITY:    storage.RuleSeverity_HIGH_RULE_SEVERITY,
	}

	log = logging.LoggerForModule()
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	// Central may have the flag on, but sensor may not.  So the pipeline
	// needs to handle both old and new results in that case.
	if features.ComplianceEnhancements.Enabled() {
		return NewPipeline(datastore.Singleton(), v2.Singleton(), scanConfigDS.Singleton())
	}

	return NewPipeline(datastore.Singleton(), nil, nil)
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(datastore datastore.DataStore, v2Datastore v2.DataStore, scanConfigDatastore scanConfigDS.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		datastore:           datastore,
		v2Datastore:         v2Datastore,
		scanConfigDatastore: scanConfigDatastore,
	}
}

type pipelineImpl struct {
	datastore           datastore.DataStore
	v2Datastore         v2.DataStore
	scanConfigDatastore scanConfigDS.DataStore
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	if features.ComplianceEnhancements.Enabled() {
		// Due to forthcoming historical result requirements, the removal of V2 results will occur through
		// pruning and retention policies as opposed to reconciliation.
		return nil
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
	if features.ComplianceEnhancements.Enabled() {
		return msg.GetEvent().GetComplianceOperatorResultV2() != nil || msg.GetEvent().GetComplianceOperatorResult() != nil
	}

	return msg.GetEvent().GetComplianceOperatorResult() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ComplianceOperatorCheckResult)

	event := msg.GetEvent()
	// If a sensor sends in a v1 compliance message we will still process it the v1 way in the event
	// a sensor is not updated or does not have the flag on.
	switch event.Resource.(type) {
	case *central.SensorEvent_ComplianceOperatorResult:
		return s.processComplianceResult(ctx, event, clusterID)
	case *central.SensorEvent_ComplianceOperatorResultV2:
		if !features.ComplianceEnhancements.Enabled() {
			return errors.New("Next gen compliance is disabled.  Message unexpected.")
		}
		return s.processV2ComplianceResult(ctx, event, clusterID)
	}

	return errors.Errorf("unexpected message %t.", event.Resource)
}

func (s *pipelineImpl) OnFinish(_ string) {}

func (s *pipelineImpl) processComplianceResult(ctx context.Context, event *central.SensorEvent, clusterID string) error {
	checkResult := event.GetComplianceOperatorResult()
	checkResult.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.datastore.Delete(ctx, event.GetId())
	default:
		return s.datastore.Upsert(ctx, checkResult)
	}
}

func (s *pipelineImpl) processV2ComplianceResult(ctx context.Context, event *central.SensorEvent, clusterID string) error {
	checkResult := event.GetComplianceOperatorResultV2()
	checkResult.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		// use V2 datastore
		return s.v2Datastore.DeleteResult(ctx, event.GetId())
	default:
		convertedResult, err := s.convertSensorMsgToV2Storage(ctx, checkResult, clusterID)
		if err != nil {
			return err
		}
		return s.v2Datastore.UpsertResult(ctx, convertedResult)
	}
}

func (s *pipelineImpl) convertSensorMsgToV2Storage(ctx context.Context, sensorData *central.ComplianceOperatorCheckResultV2, clusterID string) (*storage.ComplianceOperatorCheckResultV2, error) {
	scanConfigs, err := s.scanConfigDatastore.GetScanConfigurations(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanName, sensorData.GetSuiteName()).ProtoQuery())
	if err != nil {
		return nil, err
	}
	if len(scanConfigs) != 1 {
		return nil, errors.Errorf("Unable to find matching scan configuration for scan %q", sensorData.GetSuiteName())
	}

	return &storage.ComplianceOperatorCheckResultV2{
		Id:           sensorData.GetId(),
		CheckId:      sensorData.GetCheckId(),
		CheckName:    sensorData.GetCheckName(),
		ClusterId:    clusterID,
		Status:       statusToV2[sensorData.GetStatus()],
		Severity:     severityToV2[sensorData.GetSeverity()],
		Description:  sensorData.GetDescription(),
		Instructions: sensorData.GetInstructions(),
		Labels:       sensorData.GetLabels(),
		Annotations:  sensorData.GetAnnotations(),
		CreatedTime:  sensorData.GetCreatedTime(),
		ScanConfigId: scanConfigs[0].GetId(),
	}, nil
}
