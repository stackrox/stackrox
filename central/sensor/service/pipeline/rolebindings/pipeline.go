package rolebindings

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	countMetrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()

	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(clusterDatastore.Singleton(), datastore.Singleton())
}

// NewPipeline returns a new instance of Pipeline for k8s role bindings
func NewPipeline(clusters clusterDatastore.DataStore, bindings datastore.DataStore) pipeline.Fragment {
	return &pipelineImpl{
		clusters:        clusters,
		bindings:        bindings,
		riskReprocessor: reprocessor.Singleton(),
	}
}

type pipelineImpl struct {
	clusters        clusterDatastore.DataStore
	bindings        datastore.DataStore
	riskReprocessor reprocessor.Loop
}

func (s *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (s *pipelineImpl) Reconcile(ctx context.Context, clusterID string, storeMap *reconciliation.StoreMap) error {
	query := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	results, err := s.bindings.Search(ctx, query)
	if err != nil {
		return err
	}

	store := storeMap.Get((*central.SensorEvent_Binding)(nil))
	err = reconciliation.Perform(store, search.ResultsToIDSet(results), "k8srolebindings", func(id string) error {
		return s.runRemovePipeline(ctx, central.ResourceAction_REMOVE_RESOURCE, &storage.K8SRoleBinding{Id: id})
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetBinding() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.RoleBinding)

	event := msg.GetEvent()
	binding := event.GetBinding()
	binding.ClusterId = clusterID

	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		return s.runRemovePipeline(ctx, event.GetAction(), binding)
	case central.ResourceAction_CREATE_RESOURCE, central.ResourceAction_UPDATE_RESOURCE, central.ResourceAction_SYNC_RESOURCE:
		return s.runGeneralPipeline(ctx, event.GetAction(), binding)
	default:
		return fmt.Errorf("Event action '%s' for k8s role binding does not exist", event.GetAction())
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runRemovePipeline(ctx context.Context, _ central.ResourceAction, event *storage.K8SRoleBinding) error {
	// Validate the the event we receive has necessary fields set.
	if err := s.validateInput(event); err != nil {
		return err
	}

	// Add/Update/Remove the k8s role binding from persistence depending on the event action.
	if err := s.bindings.RemoveRoleBinding(ctx, event.GetId()); err != nil {
		return err
	}

	return nil
}

func enrichSubjects(binding *storage.K8SRoleBinding) {
	for _, subject := range binding.GetSubjects() {
		subject.ClusterId = binding.GetClusterId()
		subject.ClusterName = binding.GetClusterName()
		subject.Id = k8srbac.CreateSubjectID(subject.GetClusterId(), subject.GetName())
	}
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) runGeneralPipeline(ctx context.Context, _ central.ResourceAction, binding *storage.K8SRoleBinding) error {
	if err := s.validateInput(binding); err != nil {
		return err
	}

	if err := s.enrichCluster(ctx, binding); err != nil {
		return err
	}

	enrichSubjects(binding)

	if err := s.bindings.UpsertRoleBinding(ctx, binding); err != nil {
		return err
	}

	return nil
}

func (s *pipelineImpl) validateInput(binding *storage.K8SRoleBinding) error {
	// validate input.
	if binding == nil {
		return errors.New("role binding must not be empty")
	}
	return nil
}

func (s *pipelineImpl) enrichCluster(ctx context.Context, binding *storage.K8SRoleBinding) error {
	binding.ClusterName = ""

	clusterName, clusterExists, err := s.clusters.GetClusterName(ctx, binding.GetClusterId())
	switch {
	case err != nil:
		log.Errorf("Couldn't get name of cluster: %v", err)
		return err
	case !clusterExists:
		log.Errorf("Couldn't find cluster '%q'", binding.GetClusterId())
		return err
	default:
		binding.ClusterName = clusterName
	}
	return nil
}

func (s *pipelineImpl) OnFinish(_ string) {}
