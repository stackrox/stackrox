package complianceoperatorsuites

import (
	"context"

	"github.com/pkg/errors"
	sDatastore "github.com/stackrox/rox/central/complianceoperator/v2/suites/datastore"
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
	return NewPipeline(sDatastore.Singleton())
}

// NewPipeline returns a new instance of Pipeline for compliance suite.
func NewPipeline(suiteDatastore sDatastore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		suiteDatastore: suiteDatastore,
	}
}

type pipelineImpl struct {
	suiteDatastore sDatastore.DataStore
}

func (_ *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (p *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	// Nothing to do in this case
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}

	existingIDs := set.NewStringSet()
	suites, err := p.suiteDatastore.GetSuitesByCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	for _, suite := range suites {
		// The UID is used for reconciliation
		existingIDs.Add(suite.GetId())
	}

	store := storeMap.Get((*central.SensorEvent_ComplianceOperatorSuite)(nil))
	return reconciliation.Perform(store, existingIDs, "complianceoperatorsuites", func(id string) error {
		return p.suiteDatastore.DeleteSuite(ctx, id)
	})
}

func (_ *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetComplianceOperatorSuite() != nil
}

// Run runs the pipeline template on the input message and returns the error
func (p *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ComplianceOperatorSuite)

	if !features.ComplianceEnhancements.Enabled() {
		return errors.New("Next gen compliance is disabled. Message unexpected.")
	}

	event := msg.GetEvent()
	suite := event.GetComplianceOperatorSuite()

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return p.suiteDatastore.DeleteSuite(ctx, suite.GetId())
	default:
		return p.suiteDatastore.UpsertSuite(ctx, internaltov2storage.ComplianceOperatorSuite(suite, clusterID))
	}
}

func (p *pipelineImpl) OnFinish(_ string) {}
