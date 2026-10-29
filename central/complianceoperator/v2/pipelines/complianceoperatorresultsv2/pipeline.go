package complianceoperatorresultsv2

import (
	"context"

	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	v2 "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	reportManager "github.com/stackrox/rox/central/complianceoperator/v2/report/manager"
	"github.com/stackrox/rox/central/convert/internaltov2storage"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	_   pipeline.Fragment = (*pipelineImpl)(nil)
	log                   = logging.LoggerForModule()
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(v2.Singleton(), clusterDatastore.Singleton(), reportManager.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(v2Datastore v2.DataStore, clusterDatastore clusterDatastore.DataStore, reportMgr reportManager.Manager) pipeline.Fragment {
	return &pipelineImpl{
		v2Datastore:      v2Datastore,
		clusterDatastore: clusterDatastore,
		reportMgr:        reportMgr,
	}
}

type pipelineImpl struct {
	v2Datastore      v2.DataStore
	clusterDatastore clusterDatastore.DataStore
	reportMgr        reportManager.Manager
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	// Not currently deleting anything for purposes of building a result history for trending
	// and look back
	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetComplianceOperatorResultV2() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ComplianceOperatorCheckResultV2)

	if !features.ComplianceEnhancements.Enabled() {
		return errors.New("Next gen compliance is disabled.  Message unexpected.")
	}

	event := msg.GetEvent()
	checkResult := event.GetComplianceOperatorResultV2()
	checkResult.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		// use V2 datastore
		return s.v2Datastore.DeleteResult(ctx, event.GetId())
	default:
		clusterName, found, err := s.clusterDatastore.GetClusterName(ctx, clusterID)
		if err != nil {
			return errors.Wrapf(err, "error getting cluster name for cluster ID: %s", clusterID)
		}
		if !found {
			return errox.NotFound.Newf("cluster with id %q does not exist", clusterID)
		}
		result := internaltov2storage.ComplianceOperatorCheckResult(checkResult, clusterID, clusterName)
		// A result reflecting a genuinely new scan run is, by definition, fresh. Note: without the
		// freshness sweeper, a result can only decay to STALE through an explicit event (a scan
		// timeout); see MarkStalePriorResultsByScanRef.
		if features.ComplianceSurfaceStaleData.Enabled() {
			result.DataFreshness = s.freshnessOnReceive(ctx, result)
		}
		if err := s.reportMgr.HandleResult(ctx, result); err != nil {
			log.Errorf("unable to handle the check result in the report manager: %v", err)
			return err
		}
		return s.v2Datastore.UpsertResult(ctx, result)
	}
}

// freshnessOnReceive decides the DataFreshness a newly received result should be stored with.
// "Received" does not always mean "new": sensor resends every existing CheckResult CR on
// reconnect regardless of whether anything changed (compliance resources are not deduplicated),
// and a misbehaving compliance operator could republish a result without a real rescan behind it.
// In either case the incoming LastStartedTime (sourced from the compliance operator's own
// last-scanned-timestamp annotation) is unchanged from what is already stored, so marking it
// CURRENT would launder stale data as fresh. There is no sweeper in this slice to correct that
// later, so the check happens here: only a LastStartedTime that is newer than what is already
// stored (or the first time this result is seen) counts as genuinely fresh. Otherwise the result's
// existing freshness is carried over unchanged, since result is a freshly built proto whose
// DataFreshness would otherwise default to UNKNOWN and wipe out the prior value on upsert.
func (s *pipelineImpl) freshnessOnReceive(ctx context.Context, result *storage.ComplianceOperatorCheckResultV2) storage.ComplianceOperatorCheckResultV2_DataFreshness {
	started := result.GetLastStartedTime()

	existing, found, err := s.v2Datastore.GetComplianceCheckResult(ctx, result.GetId())
	if err != nil {
		log.Errorf("unable to look up existing compliance check result %s to evaluate freshness: %v", result.GetId(), err)
		return storage.ComplianceOperatorCheckResultV2_UNKNOWN
	}
	if !found {
		// A result with no prior history cannot be a stale replay.
		if started == nil {
			return storage.ComplianceOperatorCheckResultV2_UNKNOWN
		}
		return storage.ComplianceOperatorCheckResultV2_CURRENT
	}

	if started != nil && (existing.GetLastStartedTime() == nil || started.AsTime().After(existing.GetLastStartedTime().AsTime())) {
		return storage.ComplianceOperatorCheckResultV2_CURRENT
	}
	return existing.GetDataFreshness()
}

func (s *pipelineImpl) OnFinish(_ string) {}
