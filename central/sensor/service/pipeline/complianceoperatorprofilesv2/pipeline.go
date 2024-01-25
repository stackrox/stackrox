package complianceoperatorprofilesv2

import (
	"context"

	"github.com/pkg/errors"
	v2Datastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
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
func NewPipeline(v2ProfileDatastore v2Datastore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		v2ProfileDatastore: v2ProfileDatastore,
	}
}

type pipelineImpl struct {
	v2ProfileDatastore v2Datastore.DataStore
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}

	existingIDs := set.NewStringSet()

	profiles, err := s.v2ProfileDatastore.GetProfilesByClusters(ctx, []string{clusterID})
	if err != nil {
		return err
	}

	for _, profile := range profiles {
		// The UID is used for reconciliation
		existingIDs.Add(profile.GetId())
	}

	store := storeMap.Get((*central.SensorEvent_ComplianceOperatorProfileV2)(nil))
	return reconciliation.Perform(store, existingIDs, "complianceoperatorprofilesv2", func(id string) error {
		return s.v2ProfileDatastore.DeleteProfileForCluster(ctx, id, clusterID)
	})
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetComplianceOperatorProfileV2() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.ComplianceOperatorProfileV2)

	if !features.ComplianceEnhancements.Enabled() {
		return errors.New("Next gen compliance is disabled.  Message unexpected.")
	}

	event := msg.GetEvent()
	profile := event.GetComplianceOperatorProfileV2()

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.v2ProfileDatastore.DeleteProfileForCluster(ctx, profile.Id, clusterID)
	default:
		return s.v2ProfileDatastore.UpsertProfile(ctx, internaltov2storage.ComplianceOperatorProfileV2(profile, clusterID))
	}
}

func (s *pipelineImpl) OnFinish(_ string) {}
