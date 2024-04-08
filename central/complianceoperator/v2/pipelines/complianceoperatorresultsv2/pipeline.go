package complianceoperatorresultsv2

import (
	"context"

	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	v2 "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	"github.com/stackrox/rox/central/convert/internaltov2storage"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/metrics"
)

var (
	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(v2.Singleton(), clusterDatastore.Singleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(v2Datastore v2.DataStore, clusterDatastore clusterDatastore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		v2Datastore:      v2Datastore,
		clusterDatastore: clusterDatastore,
	}
}

type pipelineImpl struct {
	v2Datastore      v2.DataStore
	clusterDatastore clusterDatastore.DataStore
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
		return s.v2Datastore.UpsertResult(ctx, internaltov2storage.ComplianceOperatorCheckResult(checkResult, clusterID, clusterName))
	}
}

func (s *pipelineImpl) OnFinish(_ string) {}
